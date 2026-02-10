load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "swag",
    srcs = [
        "convert.go",
        "convert_types.go",
        "doc.go",
        "file.go",
        "initialism_index.go",
        "json.go",
        "loading.go",
        "name_lexem.go",
        "net.go",
        "path.go",
        "split.go",
        "string_bytes.go",
        "util.go",
        "yaml.go",
    ],
    importmap = "github.com/fasterci/rules_gitops/vendor/github.com/go-openapi/swag",
    importpath = "github.com/go-openapi/swag",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/github.com/mailru/easyjson/jlexer",
        "//vendor/github.com/mailru/easyjson/jwriter",
        "//vendor/gopkg.in/yaml.v3:yaml_v3",
    ],
)

alias(
    name = "go_default_library",
    actual = ":swag",
    visibility = ["//visibility:public"],
)
