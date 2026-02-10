load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "buffer",
    srcs = ["pool.go"],
    importmap = "github.com/fasterci/rules_gitops/vendor/github.com/mailru/easyjson/buffer",
    importpath = "github.com/mailru/easyjson/buffer",
    visibility = ["//visibility:public"],
)

alias(
    name = "go_default_library",
    actual = ":buffer",
    visibility = ["//visibility:public"],
)
