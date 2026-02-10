load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "transform",
    srcs = ["transform.go"],
    importmap = "github.com/fasterci/rules_gitops/vendor/golang.org/x/text/transform",
    importpath = "golang.org/x/text/transform",
    visibility = ["//visibility:public"],
)

alias(
    name = "go_default_library",
    actual = ":transform",
    visibility = ["//visibility:public"],
)
