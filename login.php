<?php
$payload = json_decode(file_get_contents("php://input"));
if (!$payload) {
    http_response_code(500);
    exit(json_encode("No hay payload"));
}
$nombre = $payload->nombre;
$palabraSecreta = $payload->palabraSecreta;
include_once "funciones.php";
$posibleJWT = login($nombre, $palabraSecreta);
if (!$posibleJWT) {
    http_response_code(401);
    exit("usuario o contraseña incorrecta");
}
echo $posibleJWT;
