<!DOCTYPE html>
<html>
<head>
    <title>学生管理系统</title>
</head>
<body>
    <h1>学生管理系统</h1>
    
    <h2>添加学生</h2>
    <form action="add_student.php" method="post">
        姓名：<input type="text" name="name"><br>
        年龄：<input type="text" name="age"><br>
        <input type="submit" value="添加学生">
    </form>
    
    <h2>学生列表</h2>
    <?php
    // 连接数据库
    $conn = new mysqli("localhost", "username", "password", "student_db");
    if ($conn->connect_error) {
        die("数据库连接失败：" . $conn->connect_error);
    }
    
    // 查询学生信息
    $sql = "SELECT * FROM students";
    $result = $conn->query($sql);
    
    if ($result->num_rows > 0) {
        echo "<ul>";
        while ($row = $result->fetch_assoc()) {
            echo "<li>" . $row["name"] . " - 年龄：" . $row["age"] . "</li>";
        }
        echo "</ul>";
    } else {
        echo "暂无学生信息。";
    }
    
    $conn->close();
    ?>
</body>
</html>
