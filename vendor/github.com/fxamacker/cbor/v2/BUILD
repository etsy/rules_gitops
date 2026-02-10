load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "cbor",
    srcs = [
        "bytestring.go",
        "cache.go",
        "common.go",
        "decode.go",
        "diagnose.go",
        "doc.go",
        "encode.go",
        "encode_map.go",
        "encode_map_go117.go",
        "simplevalue.go",
        "stream.go",
        "structfields.go",
        "tag.go",
        "valid.go",
    ],
    importmap = "github.com/fasterci/rules_gitops/vendor/github.com/fxamacker/cbor/v2",
    importpath = "github.com/fxamacker/cbor/v2",
    visibility = ["//visibility:public"],
    deps = ["//vendor/github.com/x448/float16"],
)

alias(
    name = "go_default_library",
    actual = ":cbor",
    visibility = ["//visibility:public"],
)
