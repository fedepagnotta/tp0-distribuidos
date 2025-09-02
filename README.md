# TP0: Docker + Comunicaciones + Concurrencia

En el presente repositorio se provee un esqueleto básico de cliente/servidor, en donde todas las dependencias del mismo se encuentran encapsuladas en containers. Los alumnos deberán resolver una guía de ejercicios incrementales, teniendo en cuenta las condiciones de entrega descritas al final de este enunciado.

El cliente (Golang) y el servidor (Python) fueron desarrollados en diferentes lenguajes simplemente para mostrar cómo dos lenguajes de programación pueden convivir en el mismo proyecto con la ayuda de containers, en este caso utilizando [Docker Compose](https://docs.docker.com/compose/).

## Instrucciones de uso

El repositorio cuenta con un **Makefile** que incluye distintos comandos en forma de targets. Los targets se ejecutan mediante la invocación de: **make \<target\>**. Los target imprescindibles para iniciar y detener el sistema son **docker-compose-up** y **docker-compose-down**, siendo los restantes targets de utilidad para el proceso de depuración.

Los targets disponibles son:

| target                | accion                                                                                                                                                                                                                                                                                                                                                                |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `docker-compose-up`   | Inicializa el ambiente de desarrollo. Construye las imágenes del cliente y el servidor, inicializa los recursos a utilizar (volúmenes, redes, etc) e inicia los propios containers.                                                                                                                                                                                   |
| `docker-compose-down` | Ejecuta `docker-compose stop` para detener los containers asociados al compose y luego `docker-compose down` para destruir todos los recursos asociados al proyecto que fueron inicializados. Se recomienda ejecutar este comando al finalizar cada ejecución para evitar que el disco de la máquina host se llene de versiones de desarrollo y recursos sin liberar. |
| `docker-compose-logs` | Permite ver los logs actuales del proyecto. Acompañar con `grep` para lograr ver mensajes de una aplicación específica dentro del compose.                                                                                                                                                                                                                            |
| `docker-image`        | Construye las imágenes a ser utilizadas tanto en el servidor como en el cliente. Este target es utilizado por **docker-compose-up**, por lo cual se lo puede utilizar para probar nuevos cambios en las imágenes antes de arrancar el proyecto.                                                                                                                       |
| `build`               | Compila la aplicación cliente para ejecución en el _host_ en lugar de en Docker. De este modo la compilación es mucho más veloz, pero requiere contar con todo el entorno de Golang y Python instalados en la máquina _host_.                                                                                                                                         |

### Servidor

Se trata de un "echo server", en donde los mensajes recibidos por el cliente se responden inmediatamente y sin alterar.

Se ejecutan en bucle las siguientes etapas:

1. Servidor acepta una nueva conexión.
2. Servidor recibe mensaje del cliente y procede a responder el mismo.
3. Servidor desconecta al cliente.
4. Servidor retorna al paso 1.

### Cliente

se conecta reiteradas veces al servidor y envía mensajes de la siguiente forma:

1. Cliente se conecta al servidor.
2. Cliente genera mensaje incremental.
3. Cliente envía mensaje al servidor y espera mensaje de respuesta.
4. Servidor responde al mensaje.
5. Servidor desconecta al cliente.
6. Cliente verifica si aún debe enviar un mensaje y si es así, vuelve al paso 2.

### Ejemplo

Al ejecutar el comando `make docker-compose-up` y luego `make docker-compose-logs`, se observan los siguientes logs:

```
client1  | 2024-08-21 22:11:15 INFO     action: config | result: success | client_id: 1 | server_address: server:12345 | loop_amount: 5 | loop_period: 5s | log_level: DEBUG
client1  | 2024-08-21 22:11:15 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°1
server   | 2024-08-21 22:11:14 DEBUG    action: config | result: success | port: 12345 | listen_backlog: 5 | logging_level: DEBUG
server   | 2024-08-21 22:11:14 INFO     action: accept_connections | result: in_progress
server   | 2024-08-21 22:11:15 INFO     action: accept_connections | result: success | ip: 172.25.125.3
server   | 2024-08-21 22:11:15 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°1
server   | 2024-08-21 22:11:15 INFO     action: accept_connections | result: in_progress
server   | 2024-08-21 22:11:20 INFO     action: accept_connections | result: success | ip: 172.25.125.3
server   | 2024-08-21 22:11:20 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°2
server   | 2024-08-21 22:11:20 INFO     action: accept_connections | result: in_progress
client1  | 2024-08-21 22:11:20 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°2
server   | 2024-08-21 22:11:25 INFO     action: accept_connections | result: success | ip: 172.25.125.3
server   | 2024-08-21 22:11:25 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°3
client1  | 2024-08-21 22:11:25 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°3
server   | 2024-08-21 22:11:25 INFO     action: accept_connections | result: in_progress
server   | 2024-08-21 22:11:30 INFO     action: accept_connections | result: success | ip: 172.25.125.3
server   | 2024-08-21 22:11:30 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°4
server   | 2024-08-21 22:11:30 INFO     action: accept_connections | result: in_progress
client1  | 2024-08-21 22:11:30 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°4
server   | 2024-08-21 22:11:35 INFO     action: accept_connections | result: success | ip: 172.25.125.3
server   | 2024-08-21 22:11:35 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°5
client1  | 2024-08-21 22:11:35 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°5
server   | 2024-08-21 22:11:35 INFO     action: accept_connections | result: in_progress
client1  | 2024-08-21 22:11:40 INFO     action: loop_finished | result: success | client_id: 1
client1 exited with code 0
```

## Parte 1: Introducción a Docker

En esta primera parte del trabajo práctico se plantean una serie de ejercicios que sirven para introducir las herramientas básicas de Docker que se utilizarán a lo largo de la materia. El entendimiento de las mismas será crucial para el desarrollo de los próximos TPs.

### Ejercicio N°1:

Definir un script de bash `generar-compose.sh` que permita crear una definición de Docker Compose con una cantidad configurable de clientes. El nombre de los containers deberá seguir el formato propuesto: client1, client2, client3, etc.

El script deberá ubicarse en la raíz del proyecto y recibirá por parámetro el nombre del archivo de salida y la cantidad de clientes esperados:

`./generar-compose.sh docker-compose-dev.yaml 5`

Considerar que en el contenido del script pueden invocar un subscript de Go o Python:

```
#!/bin/bash
echo "Nombre del archivo de salida: $1"
echo "Cantidad de clientes: $2"
python3 mi-generador.py $1 $2
```

En el archivo de Docker Compose de salida se pueden definir volúmenes, variables de entorno y redes con libertad, pero recordar actualizar este script cuando se modifiquen tales definiciones en los sucesivos ejercicios.

### Ejercicio N°2:

Modificar el cliente y el servidor para lograr que realizar cambios en el archivo de configuración no requiera reconstruír las imágenes de Docker para que los mismos sean efectivos. La configuración a través del archivo correspondiente (`config.ini` y `config.yaml`, dependiendo de la aplicación) debe ser inyectada en el container y persistida por fuera de la imagen (hint: `docker volumes`).

### Ejercicio N°3:

Crear un script de bash `validar-echo-server.sh` que permita verificar el correcto funcionamiento del servidor utilizando el comando `netcat` para interactuar con el mismo. Dado que el servidor es un echo server, se debe enviar un mensaje al servidor y esperar recibir el mismo mensaje enviado.

En caso de que la validación sea exitosa imprimir: `action: test_echo_server | result: success`, de lo contrario imprimir:`action: test_echo_server | result: fail`.

El script deberá ubicarse en la raíz del proyecto. Netcat no debe ser instalado en la máquina _host_ y no se pueden exponer puertos del servidor para realizar la comunicación (hint: `docker network`). `

### Ejercicio N°4:

Modificar servidor y cliente para que ambos sistemas terminen de forma _graceful_ al recibir la signal SIGTERM. Terminar la aplicación de forma _graceful_ implica que todos los _file descriptors_ (entre los que se encuentran archivos, sockets, threads y procesos) deben cerrarse correctamente antes que el thread de la aplicación principal muera. Loguear mensajes en el cierre de cada recurso (hint: Verificar que hace el flag `-t` utilizado en el comando `docker compose down`).

## Parte 2: Repaso de Comunicaciones

Las secciones de repaso del trabajo práctico plantean un caso de uso denominado **Lotería Nacional**. Para la resolución de las mismas deberá utilizarse como base el código fuente provisto en la primera parte, con las modificaciones agregadas en el ejercicio 4.

### Ejercicio N°5:

Modificar la lógica de negocio tanto de los clientes como del servidor para nuestro nuevo caso de uso.

#### Cliente

Emulará a una _agencia de quiniela_ que participa del proyecto. Existen 5 agencias. Deberán recibir como variables de entorno los campos que representan la apuesta de una persona: nombre, apellido, DNI, nacimiento, numero apostado (en adelante 'número'). Ej.: `NOMBRE=Santiago Lionel`, `APELLIDO=Lorca`, `DOCUMENTO=30904465`, `NACIMIENTO=1999-03-17` y `NUMERO=7574` respectivamente.

Los campos deben enviarse al servidor para dejar registro de la apuesta. Al recibir la confirmación del servidor se debe imprimir por log: `action: apuesta_enviada | result: success | dni: ${DNI} | numero: ${NUMERO}`.

#### Servidor

Emulará a la _central de Lotería Nacional_. Deberá recibir los campos de la cada apuesta desde los clientes y almacenar la información mediante la función `store_bet(...)` para control futuro de ganadores. La función `store_bet(...)` es provista por la cátedra y no podrá ser modificada por el alumno.
Al persistir se debe imprimir por log: `action: apuesta_almacenada | result: success | dni: ${DNI} | numero: ${NUMERO}`.

#### Comunicación:

Se deberá implementar un módulo de comunicación entre el cliente y el servidor donde se maneje el envío y la recepción de los paquetes, el cual se espera que contemple:

- Definición de un protocolo para el envío de los mensajes.
- Serialización de los datos.
- Correcta separación de responsabilidades entre modelo de dominio y capa de comunicación.
- Correcto empleo de sockets, incluyendo manejo de errores y evitando los fenómenos conocidos como [_short read y short write_](https://cs61.seas.harvard.edu/site/2018/FileDescriptors/).

### Ejercicio N°6:

Modificar los clientes para que envíen varias apuestas a la vez (modalidad conocida como procesamiento por _chunks_ o _batchs_).
Los _batchs_ permiten que el cliente registre varias apuestas en una misma consulta, acortando tiempos de transmisión y procesamiento.

La información de cada agencia será simulada por la ingesta de su archivo numerado correspondiente, provisto por la cátedra dentro de `.data/datasets.zip`.
Los archivos deberán ser inyectados en los containers correspondientes y persistido por fuera de la imagen (hint: `docker volumes`), manteniendo la convencion de que el cliente N utilizara el archivo de apuestas `.data/agency-{N}.csv` .

En el servidor, si todas las apuestas del _batch_ fueron procesadas correctamente, imprimir por log: `action: apuesta_recibida | result: success | cantidad: ${CANTIDAD_DE_APUESTAS}`. En caso de detectar un error con alguna de las apuestas, debe responder con un código de error a elección e imprimir: `action: apuesta_recibida | result: fail | cantidad: ${CANTIDAD_DE_APUESTAS}`.

La cantidad máxima de apuestas dentro de cada _batch_ debe ser configurable desde config.yaml. Respetar la clave `batch: maxAmount`, pero modificar el valor por defecto de modo tal que los paquetes no excedan los 8kB.

Por su parte, el servidor deberá responder con éxito solamente si todas las apuestas del _batch_ fueron procesadas correctamente.

### Ejercicio N°7:

Modificar los clientes para que notifiquen al servidor al finalizar con el envío de todas las apuestas y así proceder con el sorteo.
Inmediatamente después de la notificacion, los clientes consultarán la lista de ganadores del sorteo correspondientes a su agencia.
Una vez el cliente obtenga los resultados, deberá imprimir por log: `action: consulta_ganadores | result: success | cant_ganadores: ${CANT}`.

El servidor deberá esperar la notificación de las 5 agencias para considerar que se realizó el sorteo e imprimir por log: `action: sorteo | result: success`.
Luego de este evento, podrá verificar cada apuesta con las funciones `load_bets(...)` y `has_won(...)` y retornar los DNI de los ganadores de la agencia en cuestión. Antes del sorteo no se podrán responder consultas por la lista de ganadores con información parcial.

Las funciones `load_bets(...)` y `has_won(...)` son provistas por la cátedra y no podrán ser modificadas por el alumno.

No es correcto realizar un broadcast de todos los ganadores hacia todas las agencias, se espera que se informen los DNIs ganadores que correspondan a cada una de ellas.

## Parte 3: Repaso de Concurrencia

En este ejercicio es importante considerar los mecanismos de sincronización a utilizar para el correcto funcionamiento de la persistencia.

### Ejercicio N°8:

Modificar el servidor para que permita aceptar conexiones y procesar mensajes en paralelo. En caso de que el alumno implemente el servidor en Python utilizando _multithreading_, deberán tenerse en cuenta las [limitaciones propias del lenguaje](https://wiki.python.org/moin/GlobalInterpreterLock).

## Condiciones de Entrega

Se espera que los alumnos realicen un _fork_ del presente repositorio para el desarrollo de los ejercicios y que aprovechen el esqueleto provisto tanto (o tan poco) como consideren necesario.

Cada ejercicio deberá resolverse en una rama independiente con nombres siguiendo el formato `ej${Nro de ejercicio}`. Se permite agregar commits en cualquier órden, así como crear una rama a partir de otra, pero al momento de la entrega deberán existir 8 ramas llamadas: ej1, ej2, ..., ej7, ej8.
(hint: verificar listado de ramas y últimos commits con `git ls-remote`)

Se espera que se redacte una sección del README en donde se indique cómo ejecutar cada ejercicio y se detallen los aspectos más importantes de la solución provista, como ser el protocolo de comunicación implementado (Parte 2) y los mecanismos de sincronización utilizados (Parte 3).

Se proveen [pruebas automáticas](https://github.com/7574-sistemas-distribuidos/tp0-tests) de caja negra. Se exige que la resolución de los ejercicios pase tales pruebas, o en su defecto que las discrepancias sean justificadas y discutidas con los docentes antes del día de la entrega. El incumplimiento de las pruebas es condición de desaprobación, pero su cumplimiento no es suficiente para la aprobación. Respetar las entradas de log planteadas en los ejercicios, pues son las que se chequean en cada uno de los tests.

La corrección personal tendrá en cuenta la calidad del código entregado y casos de error posibles, se manifiesten o no durante la ejecución del trabajo práctico. Se pide a los alumnos leer atentamente y **tener en cuenta** los criterios de corrección informados [en el campus](https://campusgrado.fi.uba.ar/mod/page/view.php?id=73393).

## Notas sobre la solución de los ejercicios

### Ejercicio N°1:

#### Interfaz y parámetros

El script se invoca en la raíz del proyecto con:

```bash
./generar-compose.sh <archivo_salida> <cantidad_clientes>
```

Ejemplo:

```bash
./generar-compose.sh docker-compose-dev.yaml 5
```

- **\$1**: nombre del archivo Compose a generar.
- **\$2**: cantidad de clientes a definir.

#### Flujo general (paso a paso)

1. **Reporte de parámetros**
   Imprime los valores recibidos para facilitar el diagnóstico:

   ```bash
   echo "Nombre del archivo de salida: $1"
   echo "Cantidad de clientes: $2"
   ```

2. **Creación/limpieza del archivo**
   Garantiza que el archivo de salida exista y, acto seguido, lo **sobrescribe** con el bloque YAML inicial (operador `>`):

   ```bash
   touch $1
   echo "name: tp0
   services:
     server:
       container_name: server
       image: server:latest
       entrypoint: python3 /main.py
       environment:
         - PYTHONUNBUFFERED=1
         - LOGGING_LEVEL=DEBUG
       networks:
         - testing_net" > $1
   ```

3. **Bucle de clientes**
   Genera **client1…clientN** con `seq` y **anexa** (operador `>>`) cada bloque al YAML:

   ```bash
   for i in $(seq 1 $2); do
       echo "
     client$i:
       container_name: client$i
       image: client:latest
       entrypoint: /client
       environment:
         - CLI_ID=$i
         - CLI_LOG_LEVEL=DEBUG
       networks:
         - testing_net
       depends_on:
         - server" >> $1
   done
   ```

   - Cada cliente:
     - Usa `container_name: client<i>` para cumplir el formato solicitado (client1, client2, …).
     - Pasa `CLI_ID=$i` y `CLI_LOG_LEVEL=DEBUG` como variables de entorno.
     - Declara dependencia de **server** con `depends_on` (orden de arranque).

4. **Definición de la red**
   Cierra el archivo con la red compartida por todos los servicios:

   ```bash
   echo "
   networks:
     testing_net:
       ipam:
         driver: default
         config:
           - subnet: 172.25.125.0/24" >> $1
   ```

   - `ipam.config.subnet` fija un rango predecible; útil para pruebas y debugging de conectividad.

#### Esquema resultante (ejemplo con 2 clientes)

```yaml
name: tp0
services:
  server:
    container_name: server
    image: server:latest
    entrypoint: python3 /main.py
    environment:
      - PYTHONUNBUFFERED=1
      - LOGGING_LEVEL=DEBUG
    networks:
      - testing_net

  client1:
    container_name: client1
    image: client:latest
    entrypoint: /client
    environment:
      - CLI_ID=1
      - CLI_LOG_LEVEL=DEBUG
    networks:
      - testing_net
    depends_on:
      - server

  client2:
    container_name: client2
    image: client:latest
    entrypoint: /client
    environment:
      - CLI_ID=2
      - CLI_LOG_LEVEL=DEBUG
    networks:
      - testing_net
    depends_on:
      - server

networks:
  testing_net:
    ipam:
      driver: default
      config:
        - subnet: 172.25.125.0/24
```

#### Evitación de short-writes en client

Se implementó la función `WriteFull`, que escribe un stream de bytes al socket utilizando un loop que chequea si todavía quedan bytes por escribir. Si efectivamente
faltan, se vuelve a escribir utilizando la función `Write`, que devuelve la cantidad de bytes escritos, pero no devuelve error si hubo un short-write.

```go
func writeFull(conn net.Conn, b []byte) error {
	for len(b) > 0 {
		n, err := conn.Write(b)
		if err != nil {
			return err
		}
		b = b[n:]
	}
	return nil
}
```

En `StartClientLoop` se utiliza esta función para escribir el mensaje.

```go
msg := fmt.Sprintf("[CLIENT %v] Message N°%v\n", c.config.ID, msgID)

if err := writeFull(c.conn, []byte(msg)); err != nil {
    log.Errorf("action: send | result: fail | client_id: %v | error: %v", c.config.ID, err)
    return
}
```

#### Evitación de short-reads y short-writes en server

El manejo de la conexión con el cliente en `__handle_client_connection` fue modificado, tal que la lectura y escritura ahora se ven así:

```python
try:
    rf = client_sock.makefile("rb")
    line = rf.readline(64 * 1024)
    if line == b"":
        raise EOFError("peer closed connection")
    msg = line.rstrip(b"\r\n").decode("utf-8")
    addr = client_sock.getpeername()
    logging.info(
        "action: receive_message | result: success | ip: %s | msg: %s",
        addr[0],
        msg,
    )
    client_sock.sendall((msg + "\n").encode("utf-8"))
except (UnicodeDecodeError, EOFError, OSError) as e:
    logging.error("action: receive_message | result: fail | error: %s", e)
finally:
    client_sock.close()
```

Se puede observar que se utiliza la función `makefile` para crear un stream de bytes para solo lectura conectado al socket, a partir del cual se leen hasta
64kB del socket con la función `readline`, que internamente hace los recv necesarios para evitar short-reads.
Por otro lado, para la escritura se utiliza `sendall`, que evita short-writes internamente, ya que garantiza enviar todo el buffer o fallar.

### Ejercicio N°2:

#### Cambios en el Dockerfile del cliente

- Se elimina la copia del `config.yaml` dentro de la imagen:
  - **Antes:** `COPY ./client/config.yaml /config.yaml`
  - **Ahora:** no se incluye el archivo de configuración en la imagen.

#### Cambios en `generar-compose.sh`

- **Servidor**: se monta el `config.ini` del host dentro del contenedor en modo **read-only**:

  ```yaml
  volumes:
    - ./server/config.ini:/config.ini:ro
  ```

- **Clientes**: se monta el `config.yaml` del host dentro de cada contenedor en modo **read-only**:

  ```yaml
  volumes:
    - ./client/config.yaml:/config.yaml:ro
  ```

#### Resultado

- La configuración queda **por fuera de la imagen** y se **inyecta** en tiempo de ejecución.
- Modificar `./server/config.ini` o `./client/config.yaml` en el host impacta inmediatamente en los contenedores al reiniciarlos, **sin reconstrucción** de imágenes.
- El montaje en `:ro` asegura que los contenedores **no** modifiquen los archivos de configuración del host.

### Ejercicio N°3:

#### Enfoque general

Se orquesta un **contenedor auxiliar** (`tester`) que contiene `netcat` y se conecta al servidor a través de una **red interna de Docker**.
De esta forma, la interacción se realiza **dentro del entorno de contenedores**, cumpliendo la restricción de no instalar herramientas en el host ni exponer puertos.

#### Flujo del script `validar-echo-server.sh`

1. **Lectura de configuración del servidor**
   Se extraen `SERVER_IP` y `SERVER_PORT` desde `./server/config.ini` con `awk`.

2. **Artefactos temporales con timestamp**
   Se generan nombres únicos para:
   - Archivo Compose temporal: `docker-compose-test-<timestamp>.yaml`
   - Directorio y Dockerfile de la imagen `tester`: `dockerfile-dir-test-<timestamp>/Dockerfile`

3. **Compose mínimo para el tester**
   Se crea un `docker-compose` con un único servicio `tester` conectado a la red `testing_net` (subred fija), suficiente para ejecutar la validación:

   ```yaml
   services:
     tester:
       image: tester:latest
       networks:
         - testing_net
   networks:
     testing_net:
       ipam:
         config:
           - subnet: 172.25.125.0/24
   ```

4. **Imagen `tester` basada en Alpine**
   Se define un Dockerfile temporal que:
   - Parte de `alpine:latest`.
   - Instala `netcat-openbsd`.
   - Mantiene el contenedor en ejecución con `CMD ["sleep", "infinity"]` para permitir `docker exec`.

5. **Despliegue y ejecución del test**
   - Se construye la imagen `tester:latest` y se levanta el servicio con `docker compose up -d`.
   - Se ejecuta, dentro del contenedor `tester`, el envío del mensaje con `netcat`:

     ```sh
     echo "test_msg_<timestamp>" | timeout 10 nc <SERVER_IP> <SERVER_PORT>
     ```

   - La **respuesta** se captura en la variable `respuesta`.

6. **Detención, baja y limpieza**
   Se detiene y elimina el stack temporal (`stop`/`down`) y se borran el Dockerfile/compose generados.

7. **Criterio de validación y salida requerida**
   - Si `respuesta` coincide exactamente con `test_msg_<timestamp>`, se imprime:

     ```
     action: test_echo_server | result: success
     ```

   - En caso contrario:

     ```
     action: test_echo_server | result: fail
     ```

#### Por qué no se exponen puertos ni se instala en el host

La comunicación se realiza **contenedor a contenedor** sobre la red `testing_net`. El contenedor `tester` actúa como cliente `netcat`, por lo que no se
requieren puertos publicados hacia el host ni instalación de herramientas fuera de Docker.

#### Elección de Alpine para la imagen temporal

Se utiliza **Alpine** por ser una base **ligera**, con un conjunto de herramientas **acotado** y suficiente para el objetivo del ejercicio.
Permite instalar `netcat-openbsd` de forma simple y mantener la imagen mínima necesaria para ejecutar la validación.
