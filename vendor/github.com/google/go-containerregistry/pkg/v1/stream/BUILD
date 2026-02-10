load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "stream",
    srcs = ["layer.go"],
    importmap = "github.com/fasterci/rules_gitops/vendor/github.com/google/go-containerregistry/pkg/v1/stream",
    importpath = "github.com/google/go-containerregistry/pkg/v1/stream",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/github.com/google/go-containerregistry/pkg/v1:pkg",
        "//vendor/github.com/google/go-containerregistry/pkg/v1/types",
    ],
)

alias(
    name = "go_default_library",
    actual = ":stream",
    visibility = ["//visibility:public"],
)
