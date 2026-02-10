load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "protoiface",
    srcs = [
        "legacy.go",
        "methods.go",
    ],
    importmap = "github.com/fasterci/rules_gitops/vendor/google.golang.org/protobuf/runtime/protoiface",
    importpath = "google.golang.org/protobuf/runtime/protoiface",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/google.golang.org/protobuf/internal/pragma",
        "//vendor/google.golang.org/protobuf/reflect/protoreflect",
    ],
)

alias(
    name = "go_default_library",
    actual = ":protoiface",
    visibility = ["//visibility:public"],
)
