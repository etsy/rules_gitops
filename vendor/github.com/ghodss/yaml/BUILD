load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "yaml",
    srcs = [
        "fields.go",
        "yaml.go",
    ],
    importmap = "github.com/fasterci/rules_gitops/vendor/github.com/ghodss/yaml",
    importpath = "github.com/ghodss/yaml",
    visibility = ["//visibility:public"],
    deps = ["//vendor/gopkg.in/yaml.v2:yaml_v2"],
)

alias(
    name = "go_default_library",
    actual = ":yaml",
    visibility = ["//visibility:public"],
)
