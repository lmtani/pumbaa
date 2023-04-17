version 1.0

task RunHelloWorkflows {
    input {
        String name
    }

    command <<<
        echo "This simulates a task output file, processing string: ~{name}" > final.txt
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

    call RunHelloWorkflows as OutFromScatter {
        input:
            name=name
    }

    scatter (i in range(3)) {
        call RunHelloWorkflows {
            input:
                name="scatter_"+i
        }
    }

    output {
        File final = OutFromScatter.hello_out
        Array[File] out = RunHelloWorkflows.hello_out
    }
}
