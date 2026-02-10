load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "mergepatch",
    srcs = [
        "errors.go",
        "util.go",
    ],
    importmap = "github.com/fasterci/rules_gitops/vendor/k8s.io/apimachinery/pkg/util/mergepatch",
    importpath = "k8s.io/apimachinery/pkg/util/mergepatch",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/k8s.io/apimachinery/pkg/util/dump",
        "//vendor/sigs.k8s.io/yaml",
    ],
)

alias(
    name = "go_default_library",
    actual = ":mergepatch",
    visibility = ["//visibility:public"],
)
