load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "editor",
    srcs = ["editor.go"],
    importmap = "github.com/fasterci/rules_gitops/vendor/github.com/google/go-containerregistry/internal/editor",
    importpath = "github.com/google/go-containerregistry/internal/editor",
    visibility = ["//vendor/github.com/google/go-containerregistry:__subpackages__"],
)

alias(
    name = "go_default_library",
    actual = ":editor",
    visibility = ["//vendor/github.com/google/go-containerregistry:__subpackages__"],
)
