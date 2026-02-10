load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "merge",
    srcs = [
        "conflict.go",
        "update.go",
    ],
    importmap = "github.com/fasterci/rules_gitops/vendor/sigs.k8s.io/structured-merge-diff/v4/merge",
    importpath = "sigs.k8s.io/structured-merge-diff/v4/merge",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/sigs.k8s.io/structured-merge-diff/v4/fieldpath",
        "//vendor/sigs.k8s.io/structured-merge-diff/v4/typed",
        "//vendor/sigs.k8s.io/structured-merge-diff/v4/value",
    ],
)

alias(
    name = "go_default_library",
    actual = ":merge",
    visibility = ["//visibility:public"],
)
