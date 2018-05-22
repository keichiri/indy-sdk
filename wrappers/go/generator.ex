defmodule Utils do
  def snake_to_camel({name1, name2}) do
    {snake_to_camel(name1), snake_to_camel(name2)}
  end
  def snake_to_camel(name) do
    name
    |> String.split("_")
    |> Enum.split(1)
    |> (fn {[first], others} ->
      first <> Enum.join(Enum.map(others, &String.capitalize/1))
    end).()
  end
end


defmodule Generator do
  @function_declaration_regex ~r/extern\s(indy.*?\));/s
  @typedef_regex ~r/typedef\s+?(?<original_type>.+?)\s+?(?<alias_type>[A-Za-z0-9_-]+?);/
  @typedef_enum_regex ~r/typedef\s+?(?<original_type>enum).*?(?<alias_type>[A-Za-z0-9_-]+?);/s


  def declarations_from_path(header_directory_path) do
    File.ls!(header_directory_path)
    |> Enum.filter(&String.ends_with?(&1, ".h"))
    |> Enum.map(fn file_name ->
      full_path = Path.join(header_directory_path, file_name)
      {file_name, full_path}
    end)
    |> Enum.map(fn {file_name, full_path} ->
      declaration_strings = parse_header_file(full_path)
      {file_name, declaration_strings}
    end)
    |> Enum.reject(fn {_, strings} ->
      strings == []
    end)
    |> Enum.map(fn {file_name, declaration_strings} ->
      declaration_map = Enum.reduce(
        declaration_strings,
        [],
        fn declaration, declaration_acc ->
          [IndyFunction.parse(declaration) | declaration_acc]
        end
      )
      {file_name, declaration_map}
    end)
    |> Enum.into(%{})
  end

  def types_from_path(header_directory_path) do
    File.ls!(header_directory_path)
    |> Enum.filter(fn file_name ->
      String.contains?(file_name, "types") or String.contains?(file_name, "mod")
    end)
    |> Enum.map(&File.read!(Path.join(header_directory_path, &1)))
    |> Enum.map(&load_types/1)
    |> Enum.reduce(%{}, &Map.merge(&1, &2))
  end

  def replace_types(function_declarations, type_map) do
    function_declarations
    |> Enum.map(fn {file_name, declarations} ->
      {file_name, Enum.map(declarations, &IndyFunction.replace_types(&1, type_map))}
    end)
    |> Enum.into(%{})
  end


  def load_types(header_file_content) do
    typedef_matches = Regex.scan(@typedef_regex, header_file_content, [capture: :all_but_first])
    enum_type_def_matches = Regex.scan(@typedef_enum_regex, header_file_content, [capture: :all_but_first])
    typedef_matches ++ enum_type_def_matches
    |> Enum.into(%{}, fn [x1, x2] -> {x2, x1} end)
  end


  def parse_header_file(path) do
    content = File.read!(path)
    Regex.scan(@function_declaration_regex, content, [capture: :all_but_first])
    |> Enum.map(fn [s] -> s end)
    |> Enum.map(&String.replace(&1, ~r/(\s+|\/\/.*?\n)/, "\s"))
    |> Enum.map(&String.replace(&1, "\n", ""))
  end

  def demo do
    original_declarations = declarations_from_path("../../libindy/include")
    types = types_from_path("../../libindy/include")
    replaced_types_declarations = replace_types(original_declarations, types)
  end
end


defmodule IndyFunction do
  @declaration_regex ~r/(?<rtype>.*?)\s(?<name>.*?)\((?<params>.*?),[\s]*?void[\s]\(.*?\*.*?\)\((?<cb_params>.*?)\)/

  defstruct name: nil, parameters: [], callback_parameters: [], return_type: nil

  def parse(declaration) do
    res = Regex.named_captures(@declaration_regex, declaration, capture: :all_names)
    params = parse_parameters(res["params"])
    callback_params = parse_parameters(res["cb_params"])
    %IndyFunction{
      return_type: res["rtype"],
      name: res["name"],
      parameters: params,
      callback_parameters: callback_params,
    }
  end

  def replace_types(function, types) do
    %{function | parameters: replace_parameter_types(function.parameters, types),
                 callback_parameters: replace_parameter_types(function.callback_parameters, types)}
  end

  defp replace_parameter_types(params, types) do
    Enum.map(params, fn {original_type, param_name} ->
      {Map.get(types, original_type, original_type), param_name}
    end)
  end

  defp parse_parameters(params) do
    String.split(params, ",", trim: true)
    |> Enum.map(&String.trim/1)
    |> Enum.map(fn s ->
      {type, [arg_name]} = Enum.split(String.split(s, " "), -1)
      {Enum.join(type, " "), arg_name}
    end)
  end
end


defmodule GoTranslator do
  @c_to_go_types %{
    "enum" => "int32",
    "int32_t" => "int32",
    "const char*" => "*C.char",
    "char*" => "*C.char",
  }

  @test_f %IndyFunction{callback_parameters: [{"int32_t", "xcommand_handle"},
     {"enum", "err"}], name: "indy_create_wallet",
    parameters: [{"int32_t", "command_handle"}, {"const char*", "pool_name"},
     {"const char*", "name"}, {"const char*", "xtype"},
     {"const char*", "config"}, {"const char*", "credentials"}],
    return_type: "indy_error_t"}


  def translate_function(indy_function) do
    {c_proxy_declaration, c_proxy_definition} = generate_c_proxy(indy_function)
    {result_declaration, result_sending, result_retrieving} = Result.generate_strings(indy_function)
    {go_callback_c_declaration, go_callback} = generate_go_callback(indy_function, result_sending)
    indy_go_function = to_go_types_and_conventions(indy_function)

  end

  def to_go_types_and_conventions(indy_function) do
    %{indy_function | name: Utils.snake_to_camel(String.trim_leading(indy_function.name, "indy_")),
                      parameters: params_c_to_go(indy_function.parameters),
                      callback_parameters: params_c_to_go(indy_function.callback_parameters)}
  end

  def generate_c_proxy(indy_function) do
    parameter_strings = Enum.map(indy_function.parameters, fn {type, name} ->
      type <> " " <> name
    end)
    params_string = Enum.join(["void *fp" | parameter_strings], ", ")
    signature_string = "int32_t #{indy_function.name}_proxy (#{params_string})"
    params_type_string =
      indy_function.parameters
      |> Enum.map(fn {type, _} -> type end)
      |> Enum.join(", ")
    params_name_string =
      indy_function.parameters
      |> Enum.map(fn {_, name} -> name end)
      |> Enum.join(", ")
    function_pointer_cast_string = "int32 (*func)(#{params_type_string}) = fp;"
    function_call_string = "return func(#{params_name_string});"
    declaration = "#{signature_string};"
    definition = "#{signature_string} {\n\t#{function_pointer_cast_string}\n\t#{function_call_string}\n}"
    {declaration, definition}
  end

  def generate_go_callback(indy_function = %{callback_parameters: [{_, first_param_name} | _ ]}, result_sending) do
    {callback_name, callback_signature} = generate_go_callback_signature(indy_function)
    export = "//export #{callback_name}"
    resolver_call = "resCh, err := resolver.DeregisterCall(#{first_param_name})"
    err_check = "if err != nil {\n\t\tpanic(fmt.Sprintf(\"Invalid handle in callback: %d\", #{first_param_name}))\n\n\t}"
    "#{export}\n#{callback_signature} {\n\t#{resolver_call}\n\t#{err_check}\n\t#{result_sending}\n}"
  end

  def generate_go_callback_signature(indy_function) do
    callback_name =
      indy_function.name
      |> String.trim_leading("indy_")
      |> Utils.snake_to_camel
      |> Kernel.<>("Callback")
      go_params = to_go_types(indy_function.callback_parameters)
      go_params_string =
        go_params
        |> Enum.map(fn {type, name} -> {name, type} end)
        |> Enum.map(fn {name, type} -> name <> " " <> type end)
        |> Enum.join(", ")
        {callback_name, "func #{callback_name} (#{go_params_string})"}
  end

  defp params_c_to_go(params) do
    params
    |> to_go_types
    |> Enum.map(&Utils.snake_to_camel/1)
  end


  defp to_go_types(params) do
    Enum.map(params, fn {type, name} ->
      {Map.fetch!(@c_to_go_types, type), name}
    end)
  end
end


defmodule Result do
  def generate_strings(%{name: name, callback_parameters: [_handle_param | params]}) do
    if length(params) > 1 do
      {
        result_definition(name, params),
        result_initialisation_and_sending(name, params),
        result_retrieval_from_channel_struct(name, params)
      }
    else
      [{param_type, param_name} = first | _] = params
      {nil, "resCh <- #{param_name}", result_retrieval_from_channel_single(param_type, param_name)}
    end
  end
  def is_multiple(callback_data_params) do
    length(callback_data_params) > 1
  end

  def result_definition(function_name, callback_data_params) do
    field_definitions =
      callback_data_params
      |> Enum.map(fn {type, field} ->
        field <> " " <> type
      end)
    "type #{result_struct_name(function_name)} struct {\n\t#{Enum.join(field_definitions, "\n\t")}\n}"
  end

  def result_initialisation_and_sending(function_name, callback_data_params) do
    attr_init_strings = Enum.map(callback_data_params, fn {_, field} ->
      field <> ": " <> field
    end)
    "resCh <- &#{result_struct_name(function_name)} {\n\t#{Enum.join(attr_init_strings, ",\n\t")},\n}"
  end

  def result_retrieval_from_channel_struct(function_name, [{_, first_param_name} | _] = callback_data_params) do
    receival = "_res := <-resCh"
    assert_type = "res := _res.(*#{result_struct_name(function_name)})"
    err_msg_fmt = "fmt.Errorf(\"Indy SDK error code: %d\", res.#{first_param_name})"
    error_check = "if res.#{first_param_name} != 0 {\n\treturn nil, #{err_msg_fmt}}\n}\n"
    "#{receival}\n\t#{assert_type}\n\t#{error_check}"
  end

  def result_retrieval_from_channel_single(param_type, param_name) do
    receival = "_res := <-resCh"
    assert_type = "res := _res.(#{param_type})"
    err_msg_fmt = "fmt.Errorf(\"Indy SDK error code: %d\", res)"
    error_check = "if res != 0 {\n\t\treturn nil, #{err_msg_fmt}}\n\t}\n"
    "#{receival}\n\t#{assert_type}\n\t#{error_check}"
  end

  def result_struct_name(function_name) do
    "#{function_name}Result"
  end

end
