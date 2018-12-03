#!/bin/sh

if [ -r "VERSION" ] ; then
  . ./VERSION
fi

: ${MAJOR:=0} ${MINOR:=0} ${PATCH:=0}
VERSION="$MAJOR.$MINOR.$PATCH"

case $1 in
  --major) echo "$MAJOR" ;;
  --shell)
    echo "MAJOR=\"$MAJOR\""
    echo "MINOR=\"$MINOR\""
    echo "PATCH=\"$PATCH\""
    ;;
  *) echo "$VERSION" ;;
esac
