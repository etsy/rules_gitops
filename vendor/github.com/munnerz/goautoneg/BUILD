load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "goautoneg",
    srcs = ["autoneg.go"],
    importmap = "github.com/fasterci/rules_gitops/vendor/github.com/munnerz/goautoneg",
    importpath = "github.com/munnerz/goautoneg",
    visibility = ["//visibility:public"],
)

alias(
    name = "go_default_library",
    actual = ":goautoneg",
    visibility = ["//visibility:public"],
)
