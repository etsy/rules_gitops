load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "httptest",
    srcs = ["httptest.go"],
    importmap = "github.com/fasterci/rules_gitops/vendor/github.com/google/go-containerregistry/internal/httptest",
    importpath = "github.com/google/go-containerregistry/internal/httptest",
    visibility = ["//vendor/github.com/google/go-containerregistry:__subpackages__"],
)

alias(
    name = "go_default_library",
    actual = ":httptest",
    visibility = ["//vendor/github.com/google/go-containerregistry:__subpackages__"],
)
