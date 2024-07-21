#!/usr/bin/env bash

# Setup
TMPDIR="$(mktemp -d)"
CHARTDIR="helm/odigos helm/odigos-crds"
TAG=${TAG#V}

prefix () {
	for i in $1; do
		echo "renaming $i to $2$1"
		mv "$i" "$2$1"
	done
}

if [ -z "$TAG" ]; then
	echo "TAG required"
	exit 1
fi

if [ -z "$GITHUB_REPOSITORY" ]; then
	echo "GITHUB_REPOSITORY required"
	exit 1
fi

if [[ $(git diff -- $CHARTDIR | wc -c) -ne 0 ]]; then
	echo "Helm chart dirty. Aborting."
	exit 1
fi

# Ignore errors because it will mostly always error locally
helm repo add odigos https://odigos-io.github.io/odigos-charts 2> /dev/null || true
git worktree add $TMPDIR gh-pages -f

# Update index with new packages
for chart in "$CHARTDIR"
do
	sed -i -E 's/0.0.0/'"${TAG}"'/' $chart/Chart.yaml
done
helm package $CHARTDIR -d $TMPDIR
pushd $TMPDIR
helm repo index . --merge index.yaml --url https://github.com/$GITHUB_REPOSITORY/releases/download/$TAG/

# The check avoids pushing the same tag twice and only pushes if there's a new entry in the index
if [[ $(git diff -G apiVersion | wc -c) -ne 0 ]]; then
	# Upload new packages
	prefix *.tgz 'test-helm-assets-'
	gh release upload -R $GITHUB_REPOSITORY $TAG $TMPDIR/*.tgz

	git add index.yaml
	git commit -m "update index with $TAG" && git push
	popd
	git fetch
else
	echo "No significant changes"
	popd
fi

# Roll back chart version changes
git checkout $CHARTDIR
git worktree remove $TMPDIR -f || echo " -> Failed to clean up temp worktree"
