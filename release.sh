for GOOS in darwin linux windows; do
    echo "Building $GOOS-$GOARCH"
    export GOOS=$GOOS
    export GOARCH=amd64
    go build -o bin/cromwell-cli-$GOOS-$GOARCH
done
