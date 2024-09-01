version 1.0

import "tasks/module1.wdl"
import "subworkflow/sub.wdl"

workflow Hello {
    call sub.Sub {}

    call module1.task1 {}
}
