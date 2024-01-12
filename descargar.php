<?php
$payload = json_decode(file_get_contents("php://input"));
if (!$payload) {
    http_response_code(500);
    echo json_encode("No hay payload");
    exit;
}
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
    $rutaAbsoluta = DIRECTORIO_SUBIDAS . DIRECTORY_SEPARATOR . $detalles->nombre;
    $nombreArchivo = $detalles->nombre; // El nombre que se le sugiere al usuario cuando guarda el archivo. Solo el nombre, NO la ruta absoluta
    $tamanio = filesize($rutaAbsoluta);
    $tamanioFragmento = 5 * (1024 * 1024); //5 MB
    header('Content-Type: application/octet-stream');
    header("Content-Transfer-Encoding: Binary");
    header("Pragma: no-cache");
    header('Content-Length: ' . $tamanio);
    header(sprintf('Content-disposition: attachment; filename="%s"', $nombreArchivo));
    // Servir el archivo en fragmentos, en caso de que el tamaño del mismo sea mayor que el tamaño del fragmento
    if ($tamanio > $tamanioFragmento) {
        $manejador = fopen($rutaAbsoluta, 'rb');

        // Mientras no lleguemos al final del archivo...
        while (!feof($manejador)) {
            // Imprime lo que regrese fread, y fread leerá N cantidad de bytes en donde N es el tamaño del fragmento
            print(@fread($manejador, $tamanioFragmento));

            ob_flush();
            flush();
        }
        // Cerrar el archivo
        fclose($manejador);
    } else {
        // Si el tamaño del archivo es menor que el del fragmento, podemos usar readfile sin problema
        readfile($rutaAbsoluta);
    }
}
