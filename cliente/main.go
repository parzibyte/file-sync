package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const ArchivoAjustes = "ajustes.json"

type Ajustes struct {
	IdArchivo        string
	UbicacionArchivo string
	JWT              string
}

type Usuario struct {
	Nombre         string `json:"nombre"`
	PalabraSecreta string `json:"palabraSecreta"`
}
type DetallesArchivo struct {
	JWT       string `json:"jwt"`
	IdArchivo string `json:"idArchivo"`
}
type DetallesArchivoDevueltosPorElServidor struct {
	Id                 int64  `json:"id"`
	Nombre             string `json:"nombre"`
	IdArchivo          string `json:"id_archivo"`
	UltimaModificacion int64  `json:"ultima_modificacion"`
	FechaSubida        string `json:"fecha_subida"`
}

const urlBase = "URL_DEL_SERVIDOR"

func archivoExiste(ruta string) bool {
	if _, err := os.Stat(ruta); os.IsNotExist(err) {
		return false
	}
	return true
}

func formatearFecha(t time.Time) string {
	return fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())
}

func sincronizar(idArchivo, ubicacionArchivo, jwt string) error {
	clienteHttp := &http.Client{}
	url := urlBase + "/sincronizar.php"

	detalles := DetallesArchivo{
		JWT:       jwt,
		IdArchivo: idArchivo,
	}
	detallesComoJson, err := json.Marshal(detalles)
	if err != nil {
		return err
	}
	peticion, err := http.NewRequest("POST", url, bytes.NewBuffer(detallesComoJson))
	if err != nil {
		return err
	}
	peticion.Header.Add("Content-Type", "application/json")
	respuesta, err := clienteHttp.Do(peticion)
	if err != nil {
		return err
	}
	defer respuesta.Body.Close()
	if respuesta.StatusCode == http.StatusOK {
		var detallesRespuesta DetallesArchivoDevueltosPorElServidor
		err := json.NewDecoder(respuesta.Body).Decode(&detallesRespuesta)
		if err != nil {
			return err
		}
		fmt.Printf("El servidor devolvió los siguientes detalles\nId archivo: %s\nÚltima modificación: %s\nSubido el: %s\n",
			detallesRespuesta.IdArchivo,
			formatearFecha(time.Unix(detallesRespuesta.UltimaModificacion, 0)),
			detallesRespuesta.FechaSubida,
		)
		if !archivoExiste(ubicacionArchivo) {
			log.Printf("El archivo local no existe, descargando...")
			return descargarArchivo(idArchivo, ubicacionArchivo, jwt)
		}
		ultimaModificacionArchivoLocal, err := fechaModificacionArchivo(ubicacionArchivo)
		if err != nil {
			return err
		}
		if ultimaModificacionArchivoLocal > detallesRespuesta.UltimaModificacion {
			// El local es más reciente, lo subimos
			log.Printf("El archivo local es más reciente, subiendo...")
			return subirArchivo(idArchivo, ubicacionArchivo, jwt)
		} else if ultimaModificacionArchivoLocal == detallesRespuesta.UltimaModificacion {
			log.Printf("El archivo local y el del servidor están sincronizados. No hace falta subir ni descargar")
			return nil
		} else {
			log.Printf("El archivo local es más antiguo, descargando...")
			return descargarArchivo(idArchivo, ubicacionArchivo, jwt)
		}
	} else if respuesta.StatusCode == http.StatusNotFound {
		log.Printf("El archivo que se intenta sincronizar no existe en el servidor. Subiendo por primera vez...")
		return subirArchivo(idArchivo, ubicacionArchivo, jwt)
	} else {
		cuerpoRespuesta, err := ioutil.ReadAll(respuesta.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("código http devuelto: %d. Respuesta: %s\n", respuesta.StatusCode, string(cuerpoRespuesta))

	}
}
func hacerLogin(nombre, palabraSecreta string) (string, error) {
	clienteHttp := &http.Client{}
	url := urlBase + "/login.php"
	usuario := Usuario{
		Nombre:         nombre,
		PalabraSecreta: palabraSecreta,
	}
	usuarioComoJson, err := json.Marshal(usuario)
	if err != nil {
		return "", err
	}
	peticion, err := http.NewRequest("POST", url, bytes.NewBuffer(usuarioComoJson))
	if err != nil {
		return "", err
	}
	peticion.Header.Add("Content-Type", "application/json")
	respuesta, err := clienteHttp.Do(peticion)
	if err != nil {
		return "", err
	}
	defer respuesta.Body.Close()
	cuerpoRespuesta, err := ioutil.ReadAll(respuesta.Body)
	if err != nil {
		return "", err
	}
	respuestaString := string(cuerpoRespuesta)
	if respuesta.StatusCode == 200 {
		return respuestaString, nil
	} else {
		return "", fmt.Errorf("código de respuesta: %d. Respuesta: %s\n", respuesta.StatusCode, respuestaString)
	}
}

func fechaModificacionArchivo(ubicacionArchivo string) (int64, error) {
	info, err := os.Stat(ubicacionArchivo)
	if err != nil {
		return 0, err
	}
	return info.ModTime().Unix(), nil
}

func descargarArchivo(idArchivo string, ubicacionArchivo string, jwt string) error {
	clienteHttp := &http.Client{}
	url := urlBase + "/descargar.php"
	detalles := DetallesArchivo{
		JWT:       jwt,
		IdArchivo: idArchivo,
	}
	detallesComoJson, err := json.Marshal(detalles)
	if err != nil {
		return err
	}
	peticion, err := http.NewRequest("POST", url, bytes.NewBuffer(detallesComoJson))
	if err != nil {
		return err
	}
	peticion.Header.Add("Content-Type", "application/json")
	respuesta, err := clienteHttp.Do(peticion)
	if respuesta.StatusCode != http.StatusOK {
		return fmt.Errorf("código de respuesta: %v\n", respuesta.StatusCode)
	}
	archivoSalida, err := os.OpenFile(ubicacionArchivo, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer archivoSalida.Close()
	_, err = io.Copy(archivoSalida, respuesta.Body)
	return err
}

func subirArchivo(idArchivo string, ubicacionArchivo string, jwt string) error {
	clienteHttp := &http.Client{}
	url := urlBase + "/subir.php"
	nombreArchivo := filepath.Base(ubicacionArchivo)
	archivo, err := os.Open(ubicacionArchivo)
	if err != nil {
		return err
	}
	defer archivo.Close()
	cuerpo := &bytes.Buffer{}
	writer := multipart.NewWriter(cuerpo)
	formulario, err := writer.CreateFormFile("archivo", nombreArchivo)
	if err != nil {
		return err
	}
	io.Copy(formulario, archivo)
	modificacion, err := fechaModificacionArchivo(ubicacionArchivo)
	if err != nil {
		return err
	}
	writer.WriteField("idArchivo", idArchivo)
	writer.WriteField("jwt", jwt)
	writer.WriteField("ultimaModificacion", fmt.Sprintf("%d", modificacion))
	err = writer.Close()
	if err != nil {
		return err
	}
	peticion, err := http.NewRequest("POST", url, cuerpo)
	if err != nil {
		return err
	}
	peticion.Header.Add("Content-Type", writer.FormDataContentType())
	respuesta, err := clienteHttp.Do(peticion)
	if err != nil {
		return err
	}
	defer respuesta.Body.Close()
	if respuesta.StatusCode == http.StatusOK {
		return nil
	} else {
		return fmt.Errorf("código de respuesta: %v\n", respuesta.StatusCode)
	}
}

func elInit() error {
	for {
		ajustes, err := recuperarAjustes()
		if err != nil {
			return err
		}
		if ajustes.IdArchivo != "" && ajustes.UbicacionArchivo != "" && ajustes.JWT != "" {
			break
		}
		if ajustes.IdArchivo == "" || ajustes.UbicacionArchivo == "" {
			fmt.Println("No has configurado el archivo que se va a sincronizar.")
			err = menuGuardarAjustes()
			if err != nil {
				return err
			}
		}
		if ajustes.JWT == "" {
			fmt.Println("No has iniciado sesión.")
			err = menuLogin()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func leerAjustesYSincronizarSiEsPosible() error {
	err := elInit()
	if err != nil {
		return err
	}
	ajustes, err := recuperarAjustes()
	if err != nil {
		return err
	}
	err = sincronizar(ajustes.IdArchivo, ajustes.UbicacionArchivo, ajustes.JWT)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	var cadena string
	err := leerAjustesYSincronizarSiEsPosible()
	if err != nil {
		fmt.Printf("Error: %#v", err)
		err = menuPrincipal()
		if err != nil {
			fmt.Printf("Error: %#v", err)
		}
	} else {
		fmt.Println("Sincronizado correctamente")
	}
	fmt.Println("Presiona ENTER")
	fmt.Scanln(&cadena)
}

func menuPrincipal() error {
	var eleccion int
	var err error
	for {
		fmt.Println(`1. Configurar archivo para sincronizar
2. Login (refrescar JWT)
3. Ver ajustes
4. Sincronizar
5. Salir
Elige: [1-5]`)
		fmt.Scanln(&eleccion)
		if eleccion == 5 {
			return nil
		}
		if eleccion >= 1 && eleccion <= 5 {
			switch eleccion {
			case 1:
				err = menuGuardarAjustes()
				break
			case 2:
				err = menuLogin()
				break
			case 3:
				err = mostrarAjustes()
				break
			case 4:
				err = leerAjustesYSincronizarSiEsPosible()
			}
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		}
	}
}

func mostrarAjustes() error {
	ajustes, err := recuperarAjustes()
	if err != nil {
		return err
	}
	fmt.Printf("Id de archivo: %s\nNombre archivo: %s\nJWT: %s\n", ajustes.IdArchivo, ajustes.UbicacionArchivo, ajustes.JWT)
	return nil
}

func menuLogin() error {
	var nombre string
	var contraseña string
	for {
		fmt.Println("Nombre de usuario: ")
		fmt.Scanln(&nombre)
		fmt.Println("Contraseña: ")
		fmt.Scanln(&contraseña)
		posibleJwt, err := hacerLogin(nombre, contraseña)
		if err != nil {
			return err
		}
		ajustes, errorObteniendoAjustes := recuperarAjustes()
		if errorObteniendoAjustes != nil {
			return err
		}
		ajustes.JWT = posibleJwt
		err = guardarAjustes(ajustes)
		if err != nil {
			return err
		}
		fmt.Printf("Autenticado correctamente\n")
		return nil
	}
}

func menuArchivos() (string, error) {
	archivos, err := ioutil.ReadDir(".")
	if err != nil {
		return "", err
	}
	var ubicacionArchivo string
	var indiceArchivo int
	for indice, file := range archivos {
		if !file.IsDir() {
			fmt.Printf("%d - '%s'\n", indice+1, file.Name())
		}
	}
	for {
		fmt.Printf("Elige el archivo: [%d-%d]\n", 1, len(archivos))
		fmt.Scanln(&indiceArchivo)
		if indiceArchivo > 0 && indiceArchivo <= len(archivos) {
			ubicacionArchivo = archivos[indiceArchivo-1].Name()
			if archivoExiste(ubicacionArchivo) {
				return ubicacionArchivo, nil
			} else {
				fmt.Println("El archivo no existe")
			}
		} else {
			fmt.Println("Opción inválida")
		}
	}
}

func menuGuardarAjustes() error {
	var idArchivo string
	ubicacionArchivo, err := menuArchivos()
	if err != nil {
		return err
	}
	for {
		fmt.Println("Dale un identificador. Vas a usar ese identificador más adelante, así que no lo pierdas: ")
		fmt.Scanln(&idArchivo)
		if idArchivo != "" {
			break
		}
	}
	ajustes, err := recuperarAjustes()
	if err != nil {
		return err
	}
	ajustes.IdArchivo = idArchivo
	ajustes.UbicacionArchivo = ubicacionArchivo
	return guardarAjustes(ajustes)
}

func guardarAjustes(ajustes Ajustes) error {
	codificado, err := json.Marshal(ajustes)
	if err != nil {
		return err
	}
	return os.WriteFile(ArchivoAjustes, codificado, 0755)
}

func recuperarAjustes() (Ajustes, error) {
	var ajustes Ajustes
	if !archivoExiste(ArchivoAjustes) {
		return ajustes, nil
	}
	contenido, err := ioutil.ReadFile(ArchivoAjustes)
	if err != nil {
		return ajustes, err
	}
	err = json.Unmarshal(contenido, &ajustes)
	return ajustes, err
}
