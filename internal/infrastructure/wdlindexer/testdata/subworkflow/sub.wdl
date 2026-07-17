version 1.0

import "../tasks/module2.wdl"

workflow Sub {
    call module2.task2 {}

    output {
        String greeting = task2.greeting
    }
}
