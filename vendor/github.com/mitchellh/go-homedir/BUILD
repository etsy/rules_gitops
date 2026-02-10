load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "go-homedir",
    srcs = ["homedir.go"],
    importmap = "github.com/fasterci/rules_gitops/vendor/github.com/mitchellh/go-homedir",
    importpath = "github.com/mitchellh/go-homedir",
    visibility = ["//visibility:public"],
)

alias(
    name = "go_default_library",
    actual = ":go-homedir",
    visibility = ["//visibility:public"],
)
