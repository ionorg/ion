#!/bin/bash
set -eux

RE='[^0-9]*\([0-9]*\)[.]\([0-9]*\)[.]\([0-9]*\)\([0-9A-Za-z-]*\)'

step="$1"
if [ -z "$1" ]
then
  step=patch
fi

base="$2"
if [ -z "$2" ]
then
  base=$(git tag 2>/dev/null| tail -n 1)
  if [ -z "$base" ]
  then
    base=0.0.0
  fi
fi

MAJOR=`echo $base | sed -e "s#$RE#\1#"`
MINOR=`echo $base | sed -e "s#$RE#\2#"`
PATCH=`echo $base | sed -e "s#$RE#\3#"`

case "$step" in
  major)
    let MAJOR+=1
    let MINOR=0
    let PATCH=0
    ;;
  minor)
    let MINOR+=1
    let PATCH=0
    ;;
  patch)
    let PATCH+=1
    ;;
esac

next="v$MAJOR.$MINOR.$PATCH"
echo "$base -> $next"

git commit --allow-empty -a -m "$next"
git tag $next
git push --atomic origin master $next

