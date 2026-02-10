load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "float16",
    srcs = ["float16.go"],
    importmap = "github.com/fasterci/rules_gitops/vendor/github.com/x448/float16",
    importpath = "github.com/x448/float16",
    visibility = ["//visibility:public"],
)

alias(
    name = "go_default_library",
    actual = ":float16",
    visibility = ["//visibility:public"],
)
