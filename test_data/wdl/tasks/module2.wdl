version 1.0

task task2 {

  command {
    echo "Hello again!"
  }
  output {
    String greeting = read_string(stdout())
  }
}
