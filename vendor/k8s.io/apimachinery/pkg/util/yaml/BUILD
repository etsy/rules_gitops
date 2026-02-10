load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "yaml",
    srcs = ["decoder.go"],
    importmap = "github.com/fasterci/rules_gitops/vendor/k8s.io/apimachinery/pkg/util/yaml",
    importpath = "k8s.io/apimachinery/pkg/util/yaml",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/k8s.io/apimachinery/pkg/util/json",
        "//vendor/sigs.k8s.io/yaml",
    ],
)

alias(
    name = "go_default_library",
    actual = ":yaml",
    visibility = ["//visibility:public"],
)
