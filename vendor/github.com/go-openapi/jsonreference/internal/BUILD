load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "internal",
    srcs = ["normalize_url.go"],
    importmap = "github.com/fasterci/rules_gitops/vendor/github.com/go-openapi/jsonreference/internal",
    importpath = "github.com/go-openapi/jsonreference/internal",
    visibility = ["//vendor/github.com/go-openapi/jsonreference:__subpackages__"],
)

alias(
    name = "go_default_library",
    actual = ":internal",
    visibility = ["//vendor/github.com/go-openapi/jsonreference:__subpackages__"],
)
