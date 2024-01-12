<?php

include_once "vendor/autoload.php";
date_default_timezone_set("America/Mexico_City");

use Firebase\JWT\JWT;
use Firebase\JWT\Key;

define("CLAVE_JWT", 'CLAVE_PARA_FIRMAR_LOS_JWT');
define("DIRECTORIO_SUBIDAS", "archivos");

function cantidadUsuarios()
{
    $bd = obtenerBd();
    $sentencia = $bd->query("SELECT COUNT(*) AS conteo FROM usuarios");
    $fila = $sentencia->fetchObject();
    return $fila->conteo;
}

function prepararPrimerUso()
{
    if (cantidadUsuarios() > 0) {
        echo "No es nuevo";
        return;
    }
    $bd = obtenerBd();
    $sentencia = $bd->prepare("INSERT INTO usuarios(nombre, palabra_secreta) VALUES (? ,?)");
    $sentencia->execute(["admin", password_hash("YOUR_PASSWORD", PASSWORD_BCRYPT)]);
    echo "Primer usuario creado";
}

function borrarArchivo($idUsuario, $idArchivo)
{
    $archivo = obtenerDetallesArchivo($idUsuario, $idArchivo);
    if (!$archivo) {
        return;
    }
    $bd = obtenerBd();
    unlink(DIRECTORIO_SUBIDAS . DIRECTORY_SEPARATOR . $archivo->nombre);
    $sentencia = $bd->prepare("DELETE FROM archivos WHERE id_usuario = ? AND id_archivo = ?");
    $sentencia->execute([$idUsuario, $idArchivo]);
}

function crearDirectorioSubidasSiNoExiste()
{
    if (!file_exists(DIRECTORIO_SUBIDAS)) {
        mkdir(DIRECTORIO_SUBIDAS);
    }
}

function decodificarToken($jwt)
{
    $decoded = JWT::decode($jwt, new Key(CLAVE_JWT, 'HS256'));
    return $decoded;
}
function obtenerBd()
{
    $baseDeDatos = new PDO("sqlite:" . __DIR__ . "/archivos.db");
    $baseDeDatos->setAttribute(PDO::ATTR_ERRMODE, PDO::ERRMODE_EXCEPTION);
    $tablas = [
        "CREATE TABLE IF NOT EXISTS usuarios(
id INTEGER PRIMARY KEY AUTOINCREMENT,
nombre TEXT NOT NULL,
palabra_secreta TEXT NOT NULL);",
        "CREATE TABLE IF NOT EXISTS archivos(
id INTEGER PRIMARY KEY AUTOINCREMENT,
id_usuario INTEGER NOT NULL,
nombre TEXT NOT NULL,
id_archivo TEXT NOT NULL,
ultima_modificacion INTEGER NOT NULL,
fecha_subida TEXT NOT NULL,
FOREIGN KEY(id_usuario) REFERENCES usuarios(id) ON UPDATE CASCADE ON DELETE CASCADE);",
    ];
    foreach ($tablas as $tabla) {
        $baseDeDatos->exec($tabla);
    }
    return $baseDeDatos;
}

function obtenerUsuarioPorNombre($nombre)
{
    $bd = obtenerBd();
    $sentencia = $bd->prepare("SELECT id, nombre, palabra_secreta FROM usuarios WHERE nombre = ? LIMIT 1");
    $sentencia->execute([$nombre]);
    return $sentencia->fetchObject();
}

function guardarArchivo($idUsuario, $nombre, $idArchivo, $ultimaModificacion, $fechaSubida)
{
    $bd = obtenerBd();
    $sentencia = $bd->prepare("INSERT INTO archivos(id_usuario, nombre, id_archivo, ultima_modificacion, fecha_subida) VALUES (?, ?, ?, ?, ?)");
    return $sentencia->execute([$idUsuario, $nombre, $idArchivo, $ultimaModificacion, $fechaSubida]);
}

function login($nombre, $palabraSecreta)
{
    $usuario = obtenerUsuarioPorNombre($nombre);
    if (!$usuario) {
        return false;
    }
    $palabraSecretaHasheada = $usuario->palabra_secreta;
    if (!password_verify($palabraSecreta, $palabraSecretaHasheada)) {
        return false;
    }
    $payload = [
        'iat' => time(),
        'nbf' => time(),
        'exp' => time() + ((30 * 24 * 60 * 60)),
        'id_usuario' => $usuario->id,
    ];
    $jwt = JWT::encode($payload, CLAVE_JWT, 'HS256');
    return $jwt;
}

function obtenerDetallesArchivo($idUsuario, $idArchivo)
{
    $bd = obtenerBd();
    $sentencia = $bd->prepare("SELECT id, nombre, id_archivo, ultima_modificacion, fecha_subida 
    FROM archivos 
    WHERE id_usuario = ? AND id_archivo = ? ORDER BY ultima_modificacion DESC LIMIT 1");
    $sentencia->execute([$idUsuario, $idArchivo]);
    return $sentencia->fetchObject();
}
