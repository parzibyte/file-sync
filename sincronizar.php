<?php
$payload = json_decode(file_get_contents("php://input"));
if (!$payload) {
    http_response_code(500);
    echo json_encode("No hay payload");
    exit;
}
/*
Recibimos solo el nombre del archivo y la última fecha de modificación
Buscamos si tenemos un registro de ese nombre de archivo junto con el id de usuario
Existe: 
    Devolvemos la fecha de última modificación más reciente, así el cliente sabrá 
    qué hacer (descargarlo o subirlo)
No existe:
    Devolvemos un 404 not found. El cliente sabrá que el archivo no existe y subirá el nuevo archivo
 */
include_once "funciones.php";
$jwt = $payload->jwt;
try {
    $jwtDecodificado = decodificarToken($jwt);
} catch (Exception $e) {
    http_response_code(401);
    echo json_encode($e->getMessage());
    exit();
}
$idUsuario = $jwtDecodificado->id_usuario;
$idArchivo = $payload->idArchivo;
$detalles = obtenerDetallesArchivo($idUsuario, $idArchivo);
if (!$detalles) {
    http_response_code(404);
    echo json_encode("No encontrado");
} else {
    echo json_encode($detalles, JSON_NUMERIC_CHECK);
}
