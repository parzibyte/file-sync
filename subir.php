<?php
if (!isset($_POST["jwt"])) {
    http_response_code(401);
    echo json_encode("Autentificación requerida");
    exit;
}
if (!isset($_POST["ultimaModificacion"])) {
    http_response_code(500);
    echo json_encode("ultimaModificacion no encontrada");
    exit;
}
if (!isset($_POST["idArchivo"])) {
    http_response_code(500);
    echo json_encode("idArchivo no encontrada");
    exit;
}
if (!isset($_FILES["archivo"])) {
    http_response_code(500);
    echo json_encode("No hay archivo");
    exit();
}
date_default_timezone_set("America/Mexico_City");
include_once "funciones.php";
$jwt = $_POST["jwt"];
$ultimaModificacion = $_POST["ultimaModificacion"];
$idArchivo = $_POST["idArchivo"];
try {
    $jwtDecodificado = decodificarToken($jwt);
} catch (Exception $e) {
    http_response_code(401);
    echo json_encode($e->getMessage());
    exit();
}
$archivo = $_FILES["archivo"];
$ahora = date("Y-m-d H:i:s");
$nombreArchivo = $archivo["name"];
$extension = pathinfo($nombreArchivo, PATHINFO_EXTENSION);
$nuevoNombre = uniqid("", true) . "." . $extension;
$nuevaUbicacion = DIRECTORIO_SUBIDAS . DIRECTORY_SEPARATOR . $nuevoNombre;
crearDirectorioSubidasSiNoExiste();
move_uploaded_file($archivo["tmp_name"], $nuevaUbicacion);
$idUsuario = $jwtDecodificado->id_usuario;
# En hostings sin límite de espacio, tal vez no deberíamos borrar el archivo para tener varias versiones
borrarArchivo($idUsuario, $idArchivo);
guardarArchivo($idUsuario, $nuevoNombre, $idArchivo, $ultimaModificacion, $ahora);
