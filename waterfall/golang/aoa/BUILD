load(
    "@io_bazel_rules_go//go:def.bzl",
    "go_library",
    "go_binary",
)

package(default_visibility = ["//visibility:public"])

# In order to build this target libusb needs to be installed.
go_library(
    name = "aoa",
    srcs = ["aoa.go"],
    deps = ["@com_github_google_gousb//:gousb"],
    importpath = "github.com/google/waterfall/golang/aoa",
    cgo = True,
)


go_binary(
    name = "aoa_utils_bin",
    srcs = ["aoa_utils_bin.go"],
    deps = [
        ":aoa",
    ],
    cgo = True,
)

