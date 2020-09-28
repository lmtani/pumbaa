for GOOS in darwin linux; do
    echo "Building $GOOS-$GOARCH"
    export GOARCH=amd64
    export GOOS=$GOOS
    go build -o bin/cromwell-cli-$GOOS-$GOARCH
done
