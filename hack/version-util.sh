# source this file in a script to call these functions

# We extract version information from git tags
# the implicit contract is that our git tags will be in ~semver (three-part) format and prefaced with the letter 'v'.
# this contract is required by the goreleaser tool and used throughout Carvel suite.

# git tag version extraction adapted from https://github.com/vmware-tanzu/carvel-imgpkg/blob/develop/hack/build-binaries.sh
function get_latest_git_tag {
  git describe --tags | grep -Eo 'v[0-9]+\.[0-9]+\.[0-9]+(-alpha\.[0-9]+)?'
}

function get_sgctrl_ver {
  echo "${1:-`get_latest_git_tag`}"
}

function get_sgctrl_ver_without_v {
  git describe --tags | grep -Eo '[0-9]+\.[0-9]+\.[0-9]+(-alpha\.[0-9]+)?'
}