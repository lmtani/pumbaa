version 1.0

task task1 {
    command {
        echo "Hello World!" > final.txt
    }
    output {
        File final = "final.txt"
    }
}
