#!/usr/bin/env bash
# Automates docs/adr/0002-cli-module-split.md's "Releasing the split modules"
# recipe: tag pkg/signature, then pkg/template, then satellites/recipes, then
# satellites/sparrow, each
# with their working-tree `replace` directives stripped and `require`s pinned
# to the version just tagged — so `go install .../satellites/sparrow@vX.Y.Z`
# (and @latest) resolves without the "go.mod contains replace directives"
# error (F-002).
#
# Each module's edit is made and tagged in its own disposable git worktree
# (never on a branch merged to main) so `main` keeps its go.work `replace`
# directives for local development, per the ADR.
#
# Usage:
#   scripts/release-submodules.sh vX.Y.Z          # dry run: print the plan only
#   scripts/release-submodules.sh vX.Y.Z --push   # tag and push each module for real
#
# Safe to re-run: any module already tagged at the requested version is skipped.
set -euo pipefail

VERSION="${1:?usage: $0 vX.Y.Z [--push]}"
PUSH=0
[[ "${2:-}" == "--push" ]] && PUSH=1

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
	echo "error: version must look like vX.Y.Z, got: $VERSION" >&2
	exit 1
fi

repo_root="$(git -C "$(dirname "${BASH_SOURCE[0]}")/.." rev-parse --show-toplevel)"
cd "$repo_root"

if [[ -n "$(git status --porcelain)" ]]; then
	echo "error: working tree is dirty; commit or stash before releasing" >&2
	exit 1
fi

tag_exists() { git rev-parse -q --verify "refs/tags/$1" >/dev/null 2>&1; }

# release_leaf tags a dependency-free module (no replace directives to strip)
# directly at the given commit-ish.
release_leaf() {
	local module_dir="$1" tag="$2"
	if tag_exists "$tag"; then
		echo "skip: $tag already exists"
		return
	fi
	echo "plan: tag $tag at HEAD ($module_dir has no replace directives to strip)"
	if [[ "$PUSH" == 1 ]]; then
		git tag -a "$tag" -m "$tag"
		git push origin "$tag"
	fi
}

# release_module strips `replace` lines from module_dir/go.mod, pins each dep
# in deps (module@version pairs) to its real released version, commits that
# in a disposable worktree, and tags the commit.
release_module() {
	local module_dir="$1" tag="$2"
	shift 2
	local deps=("$@") # e.g. github.com/sarathsp06/sparrow/pkg/signature@v1.2.3

	if tag_exists "$tag"; then
		echo "skip: $tag already exists"
		return
	fi

	echo "plan: strip replace directives in $module_dir/go.mod, pin ${deps[*]}, tag $tag"
	if [[ "$PUSH" != 1 ]]; then
		return
	fi

	local wt
	wt="$(mktemp -d)/wt"
	git worktree add -q --detach "$wt" HEAD

	(
		export GOWORK=off
		trap 'cd "$repo_root"; git worktree remove --force "$wt" >/dev/null 2>&1 || true' EXIT
		cd "$wt/$module_dir"
		for dep in "${deps[@]}"; do
			go mod edit -dropreplace="${dep%@*}" -require="$dep"
		done
		go mod tidy
		cd "$wt"
		git add "$module_dir/go.mod" "$module_dir/go.sum"
		git commit -q -m "chore(release): $tag — strip workspace replace for release"
		git tag -a "$tag" -m "$tag"
		git push origin "$tag"
	)
}

release_leaf pkg/signature "pkg/signature/$VERSION"
release_module pkg/template "pkg/template/$VERSION" \
	"github.com/sarathsp06/sparrow/pkg/signature@$VERSION"
release_module satellites/recipes "satellites/recipes/$VERSION" \
	"github.com/sarathsp06/sparrow/pkg/template@$VERSION"
release_module satellites/sparrow "satellites/sparrow/$VERSION" \
	"github.com/sarathsp06/sparrow/pkg/signature@$VERSION" \
	"github.com/sarathsp06/sparrow/pkg/template@$VERSION" \
	"github.com/sarathsp06/sparrow/satellites/recipes@$VERSION"

if [[ "$PUSH" != 1 ]]; then
	echo
	echo "dry run only — re-run with --push to tag and push for real."
fi
