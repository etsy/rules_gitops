load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "pointer",
    srcs = ["pointer.go"],
    importmap = "github.com/fasterci/rules_gitops/vendor/k8s.io/utils/pointer",
    importpath = "k8s.io/utils/pointer",
    visibility = ["//visibility:public"],
    deps = ["//vendor/k8s.io/utils/ptr"],
)

alias(
    name = "go_default_library",
    actual = ":pointer",
    visibility = ["//visibility:public"],
)
