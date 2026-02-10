load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "static",
    srcs = ["layer.go"],
    importmap = "github.com/fasterci/rules_gitops/vendor/github.com/google/go-containerregistry/pkg/v1/static",
    importpath = "github.com/google/go-containerregistry/pkg/v1/static",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/github.com/google/go-containerregistry/pkg/v1:pkg",
        "//vendor/github.com/google/go-containerregistry/pkg/v1/types",
    ],
)

alias(
    name = "go_default_library",
    actual = ":static",
    visibility = ["//visibility:public"],
)
