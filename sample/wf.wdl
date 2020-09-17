version 1.0

task RunHelloWorkflows {
    input {
        String name
    }

    command <<<
        echo "This simulates a task output file, processig string: ~{name}" > final.txt
    >>>

    output {
        File hello_out = "final.txt"
    }
}


workflow HelloWorld {
    input {
        String name
    }

    call RunHelloWorkflows {
        input:
            name=name
    }

    output {
        String final = RunHelloWorkflows.hello_out
    }
}
