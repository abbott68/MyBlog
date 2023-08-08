<?php
if ($_SERVER["REQUEST_METHOD"] == "POST") {
    $name = $_POST["name"];
    $age = $_POST["age"];
    
    $conn = new mysqli("localhost", "username", "password", "student_db");
    if ($conn->connect_error) {
        die("数据库连接失败：" . $conn->connect_error);
    }
    
    $sql = "INSERT INTO students (name, age) VALUES ('$name', $age)";
    
    if ($conn->query($sql) === TRUE) {
        echo "学生信息已添加成功。";
    } else {
        echo "添加学生信息失败：" . $conn->error;
    }
    
    $conn->close();
}
?>
