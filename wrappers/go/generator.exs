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
    IO.inspect original_declarations
    types = types_from_path("../../libindy/include")
    IO.inspect types
    replaced_types_declarations = replace_types(original_declarations, types)
    IO.inspect replaced_types_declarations
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
    IO.puts "Types: #{inspect types}"
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


defmodule Translator do
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
    "#{signature_string} {\n\t#{function_pointer_cast_string}\n\t#{function_call_string}\n}"
  end

  def generate_go_callback(indy_function) do
    go_params = to_go_types(indy_function.callback_params)
    go_params_string =
      go_params
      |> Enum.map(fn {type, name} -> {name, type} end)
      |> Enum.join(", ")

  end


  defp to_go_types(params) do
    Enum.map(params, fn {type, name} ->
      {Map.fetch!(@c_to_go_types, type), name}
    end)
  end

  def generate_go_callback_signature(indy_function) do
    callback_name =
      indy_function.name
      |> String.trim_leading("indy_")
      |> String.split("_")
      |> Enum.split(1)
      |> (fn {[first], rest} ->
        Enum.join([first | Enum.map(rest, &String.capitalize/1)], "")
      end).()
    IO.puts("Callback name: #{inspect callback_name}")
    go_params = to_go_types(indy_function.callback_parameters)
    go_params_string =
      go_params
      |> Enum.map(fn {type, name} -> {name, type} end)
      |> Enum.map(fn {name, type} -> name <> " " <> type end)
      |> Enum.join(", ")
    "func #{callback_name}(#{go_params_string})"
  end

end
