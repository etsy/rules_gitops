load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "yaml",
    srcs = [
        "fields.go",
        "yaml.go",
        "yaml_go110.go",
    ],
    importmap = "github.com/fasterci/rules_gitops/vendor/sigs.k8s.io/yaml",
    importpath = "sigs.k8s.io/yaml",
    visibility = ["//visibility:public"],
    deps = ["//vendor/sigs.k8s.io/yaml/goyaml.v2:goyaml_v2"],
)

alias(
    name = "go_default_library",
    actual = ":yaml",
    visibility = ["//visibility:public"],
)
