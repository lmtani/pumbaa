version 1.0

task RunHelloWorkflows {
    input {
        String name
    }

    command <<<
        echo "This simulates a task output file, processig string: ~{name}" > final.txt
    >>>

    runtime {
        docker: "ubuntu:20.04"
    }

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
        File final = RunHelloWorkflows.hello_out
    }
}
