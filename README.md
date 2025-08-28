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

### Ejercicio N°4:

#### Server

Se utilizó un nuevo parámetro `_running` en la clase `Server`, que se inicializa en `False`, y en el método `Server.run()` se setea a `True`. La condición del `while`
pasa a chequear ese parámetro, de tal forma que, al convertirse en `False`, se dejan de aceptar nuevas conexiones.

Para cambiar el valor de `_running` se creó un handler para la señal `SIGTERM`. En este handler, llamado `__stop_running`, se setea `_running` a `False`, y se cierra
el `socket` del server (`Server._server_socket`). Al cerrar el `socket` se evita que el el proceso quede bloqueado esperando una nueva conexión de un cliente, ya que se
podría dar que se chequea la condición del `while`, se cumple esa condición, se pasa a esperar una nueva conexión, y recién ahí llega un `SIGTERM`. Cerrando el socket
hacemos que falle el `accept()` con `OSError`, y ahí se chequea si `self._running == False`, y en ese caso se rompe el loop en vez de propagar el error.

Cuando se sale del loop principal de `run`, se llama a `logging.shutdown()` para flushear buffers y liberar recursos utilizados para logging.

En resumen: al recibir `SIGTERM`, se drena la última conexión establecida con un cliente y se dejan de aceptar conexiones nuevas.

Versión final de `run()`:

```python
    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """

        self._running = True
        signal.signal(signal.SIGTERM, self.__stop_running)
        while self._running:
            try:
                client_sock = self.__accept_new_connection()
                self.__handle_client_connection(client_sock)
            except OSError:
                if not self._running:
                    break
                raise
        logging.shutdown()
```

Versión final de `__stop_running()`:

```python
    def __stop_running(self, signum, frame):
        self._running = False
        self._server_socket.close()
```

#### Client

Se utilizó un `context` creado con `signal.NotifyContext`, para "bypassear" el comportamiento por defecto de Go a la hora de recibir señales `SIGTERM`. En otras palabras,
`signal.NotifyContext` convierte `SIGTERM` en cancelación de contexto, lo que permite handlear la interrupción, limpiando lo necesario y terminando de forma ordenada.

Este contexto se creó en la función `StartClientLoop`. Seguido de esto, se utilizó `defer stop()` para posponer la interrupción del programa hasta que la función
retorne. De esta forma, se puede handlear la señal, hacer un `graceful shutdown`, y finalmente terminar la ejecución.

El loop de `StartClientLoop` fue modificado, quedando así:

```go
for msgID := 1; msgID <= c.config.LoopAmount; msgID++ {
    select {
    case <-ctx.Done():
        return
    default:
    }
    // Create the connection the server in every loop iteration. Send an
    c.createClientSocket()

    // TODO: Modify the send to avoid short-write
    fmt.Fprintf(
        c.conn,
        "[CLIENT %v] Message N°%v\n",
        c.config.ID,
        msgID,
    )

    readDone := make(chan struct{})
    var msg string
    var err error
    go func() {
        msg, err = bufio.NewReader(c.conn).ReadString('\n')
        close(readDone)
    }()

    timer := time.NewTimer(c.config.LoopPeriod)
    select {
    case <-ctx.Done():
        _ = c.conn.SetReadDeadline(time.Now())
        <-readDone
        timer.Stop()
        c.conn.Close()
        return
    case <-timer.C:
        select {
        case <-ctx.Done():
            _ = c.conn.SetReadDeadline(time.Now())
            <-readDone
            timer.Stop()
            c.conn.Close()
            return
        case <-readDone:
            c.conn.Close()
            if err != nil {
                log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
                    c.config.ID,
                    err,
                )
                return
            }
            log.Infof("action: receive_message | result: success | client_id: %v | msg: %v",
                c.config.ID,
                msg,
            )
        }
    }
}
```

Veámoslo en detalle:

1. Al principio de cada iteración se agregó un chequeo, para ver si la señal ya había sido notificada en el contexto.
2. La lectura del buffer se pasó a una `goroutine`, que cierra un canal `readDone` al finalizar la lectura.
3. En vez de hacer un `sleep` para separar los envíos del cliente, se utilizó un `timer`, y luego, utilizando un `select`, la main routine quedó bloqueada
   esperando la señal del contexto (el `SIGTERM`), o la señal del timer (la finalización del `LoopPeriod` seteado en el archivo de configuración).

- Si lo primero en suceder es que llega el `SIGTERM`, se setea un deadline para la operación de lectura que estaba realizando la goroutine, haciendo que se interrumpa
  la misma. Luego, se espera a que se cierre el canal `readDone` para frenar el `timer`, cerrar el `socket` del cliente, y retornar para ejecutar el `stop()` del `defer`.
- Si lo primero en suceder es que se termina el timer, se vuelve a hacer un `select`, esta vez esperando al `SIGTERM` o a que se finalice la lectura de la `goroutine`.
  Si llega el `SIGTERM` primero, se ejecuta el flujo mencionado justo arriba. En caso contrario, simplemente se cierra el socket de la iteración actual, y se loggea el
  resultado correspondiente.

De esta forma, tanto la lectura como la espera entre envíos al servidor son interrumpibles de forma ordenada, y se puede realizar un `graceful shutdown` al recibir un
`SIGTERM`.

### Ejercicio N°5:

#### Protocolo

El protocolo de comunicación utilizado se basa en envío de paquetes con ordenamiento little endian y el siguiente formato:

| opcode | length | body |
0........1........5......5 + length

Para describir el formato del cuerpo de los paquetes, utilizaremos la siguiente notación (basada en el protocolo nativo de Cassandra):

- [int]: Un entero de 4 bytes.
- [string]: Un n de tipo [int], seguido de n bytes representando un string UTF-8.
- [string map]: Un n de tipo [int], seguido de n pares <k><v> donde <k> y <v> son [string].
- [multi string map]: Un n de tipo [int], seguido de n [string map].

Como se puede observar, el campo `opcode` es el primer byte del paquete, y puede tomar los siguientes valores:

- 0: _NEW_BETS_. Es enviado por el cliente, y representa un conjunto de apuestas. Es el único mensaje que tiene un body, y éste es un [multi string map]. En este ejercicio
  este [multi string map] va a tener tamaño fijo 1, pero se utiliza este formato para favorecer escalabilidad a futuro y facilitar la implementación de envío
  de bets por batches. Dado que la única diferencia entre un [string map] y un [multi string map] de tamaño 1 son 4 bytes, se considera que el trade off es
  positivo, y vale la pena simplificar implementaciones futuras en desmero de la ligereza de los paquetes, ya que la diferencia de tamaño en bytes es muy pequeña.

- 1: _BETS_RECV_SUCCESS_. Es enviado por el server en respuesta al cliente si pudo procesar con éxito todas las apuestas.

- 2: _BETS_RECV_FAIL_. Es enviado por el server en respuesta al cliente si hubo un error al procesar alguna de las apuestas.

Por último, el campo `length` indica la longitud total en bytes del body.

#### Client-side

**Resumen de la implementación**

El cliente está dividido en dos módulos:

- `client/common/communication.go`: define el **protocolo**, los **mensajes**, y la **serialización**/**deserialización** (capa de transporte).
- `client/common/client.go`: maneja la **conexión**, el **ciclo de envío/recepción**, el **manejo de señales** y el **logging** (capa de aplicación).

Esta separación cumple con la consigna de “correcta separación de responsabilidades”: el modelo de dominio de los mensajes y su codificación vive en `communication.go`,
mientras que la lógica de negocio del cliente (abrir socket, enviar apuesta, esperar confirmación y loguear) vive en `client.go`.

---

**Aspectos clave de la implementación**

1. **Definición de un protocolo para el envío de los mensajes**
   - El formato físico del paquete es `opcode (1 byte) | length (int32 LE) | body`, con ordenamiento **little endian**, tal como se documenta en la sección de Protocolo.
   - Se modelan tipos de mensaje concretos: `NewBets` (cliente→servidor), `BetsRecvSuccess` y `BetsRecvFail` (servidor→cliente).
     Los mensajes implementan interfaces (`Message`, `Writeable`, `Readable`) para dejar explícita la responsabilidad de cada uno (obtener opcode, escribirse/leerse).

2. **Serialización de los datos**
   - La función `writeString` serializa strings como `[int32 longitud][bytes UTF-8]`.
   - `writePair` y `writeMultiStringMap` construyen el **\[string map]** y el **\[multi string map]** (con su contador `int32` previo),
     exactamente como se definió en el protocolo.
   - `NewBets.WriteTo` arma el body en un `bytes.Buffer`, antepone el `length` y luego compone el paquete completo. Esto garantiza que el paquete enviado
     respeta la estructura pactada.
   - En la lectura, `ReadMessage` consume primero el `opcode` y, según su valor, delega en el `readFrom` específico del mensaje.
     En las respuestas del servidor (`BetsRecvSuccess`/`BetsRecvFail`) se valida que `length == 0`, lo que agrega **sanidad de protocolo** (si llega basura, se rechaza).

3. **Correcta separación de responsabilidades**
   - **Transporte** (`communication.go`): sabe serializar/deserializar y validar mínimos de protocolo (opcodes válidos, longitudes esperadas).
     Define `ProtocolError` con contexto (incluye `Opcode`) para facilitar diagnóstico.
   - **Aplicación** (`client.go`): abre el socket (`createClientSocket`), compone el mensaje `NewBets` con los datos de la apuesta y maneja tiempos de vida/cierre,
     errores y logging.

4. **Empleo correcto de sockets, manejo de errores y evitación de _short read_ / _short write_**
   - **Evita _short write_**:
     - `NewBets.WriteTo` construye el paquete completo en memoria y usa `io.Copy(out, &buff)`.
     - `io.Copy`/`bytes.Buffer.WriteTo` **reintenta internamente** hasta transferir todos los bytes o fallar, cubriendo _partial writes_
       del `net.Conn` sin que el llamador tenga que implementar el bucle manual.
     - Al devolver la longitud escrita y propagar errores, se permite loguear/actuar ante fallas.

   - **Evita _short read_**:
     - La lectura usa un `bufio.Reader` y **operaciones de tamaño fijo**: `ReadByte` (para `opcode`) y `binary.Read` (para `int32 length`),
       que leen **exactamente** el número de bytes requerido o devuelven error.
     - Al validar que `length == 0` en las respuestas, se evita intentar leer un body ausente y se detectan desalineaciones
       (previniendo lecturas incompletas o corridas).

   - **No bloqueo indefinido y cierre ordenado**:
     - `SendBet` atacha un contexto a `SIGTERM` con `signal.NotifyContext`. La lectura del mensaje de respuesta se hace en una goroutine y se
       coordina con un canal `readDone`.
     - Si llega una señal, se fuerza un `SetReadDeadline(time.Now())` para **desbloquear** la goroutine de lectura y poder cerrar el `conn`
       limpiamente (**graceful shutdown**).
     - En ambos caminos (respuesta o cancelación), el socket se cierra de forma explícita.

5. **Variables de entorno y contenido de la apuesta**
   - La implementación de `SendBet(name, lastName, dni, birthDate, number)` **recibe** exactamente los campos `NOMBRE`, `APELLIDO`, `DOCUMENTO`, `NACIMIENTO` y
     `NUMERO` y arma un `NewBets` con un único **\[string map]** que incluye:
     - `AGENCIA`: `c.config.ID` (identifica a la agencia; satisface el requisito de 5 agencias distintas configurando IDs distintos).
     - `NOMBRE`, `APELLIDO`, `DOCUMENTO`, `NACIMIENTO`, `NUMERO`: con los valores suministrados (leídos del entorno en el `main`).

   - Se modificó el script `generar-compose.sh` para inyectar dichos valores en el `docker-compose` vía `environment:` y el
     proceso que invoca `SendBet` los pasa como parámetros. De este modo, se cumple la interfaz pedida por el enunciado sin acoplar
     la capa de transporte a `os.Getenv`.

6. **Confirmación y logging conforme al enunciado**
   - Tras enviar `NewBets`, el cliente **espera una única respuesta**: `BETS_RECV_SUCCESS` u `BETS_RECV_FAIL`.
   - Si llega `BETS_RECV_SUCCESS`, se loguea exactamente:

     ```
     action: apuesta_enviada | result: success | dni: ${DNI} | numero: ${NUMERO}
     ```

   - Si hay error de I/O, `opcode` inválido, `length` no esperado, o respuesta `BETS_RECV_FAIL`, se loguea el caso `fail` con los
     mismos campos de contexto (`dni`, `numero`).
   - Se usa `go-logging` y se reportan también errores críticos de conexión (por ejemplo, en `createClientSocket`).

---

**Detalles de diseño relevantes**

- **Ordenamiento de claves en los mapas**: en Go, iterar un `map[string]string` no garantiza orden; esto **no afecta** la interoperabilidad porque
  el protocolo serializa **pares `<k><v>` auto-descriptivos** con un contador previo. El servidor reconstruye el diccionario sin asumir orden.
- **Tamaños y tipos**: se usan `int32` para longitudes/contadores, suficientes para los volúmenes del TP y homogéneos con la notación del protocolo.

#### Server-side

**Resumen de la implementación**

El servidor se organiza en módulos con responsabilidades bien delimitadas:

- `server/common/communication.py`: capa de **protocolo/transporte**. Define opcodes, formato binario (little endian), rutinas de serialización/deserialización,
  validaciones estructurales y los mensajes `BETS_RECV_SUCCESS`/`BETS_RECV_FAIL`. Implementa la lectura robusta del stream y el procesamiento de `NEW_BETS`.
- `server/common/server.py`: capa de **aplicación**. Acepta conexiones, coordina el ciclo petición/respuesta, maneja señales (graceful shutdown),
  centraliza logging y cierre de recursos.
- `server/common/utils.py`: **persistencia** y modelo de dominio (`Bet`, `store_bets`, `has_won`, etc.).
- `server/main.py`: **bootstrap** (parsing de configuración desde variables de entorno/archivo, inicialización de logging y arranque del loop del servidor).

---

**Aspectos clave de la implementación**

1. **Definición del protocolo para el envío de los mensajes**
   - Los opcodes reflejan exactamente la especificación: `NEW_BETS=0`, `BETS_RECV_SUCCESS=1`, `BETS_RECV_FAIL=2` (clase `Opcodes`).
   - El framing del paquete es `opcode (u8) | length (i32 LE) | body`, con endianness **little endian** (todos los `read_*`/`write_*` utilizan formatos `"<B"`, `"<i"`).
   - Para `NEW_BETS`, el body es un **\[multi string map]**: primero un `int32` con el número de apuestas, y luego por cada apuesta un
     **\[string map]** de exactamente 6 pares `<k><v>` (`AGENCIA`, `NOMBRE`, `APELLIDO`, `DOCUMENTO`, `NACIMIENTO`, `NUMERO`). La clase `NewBets`
     valida ambos aspectos (cantidad de pares y presencia de claves requeridas).

2. **Serialización/Deserialización de datos**
   - **Deserialización (entrada):**
     - `recv_msg` lee `opcode` (`read_u8`) y `length` (`read_i32` con verificación de remanente). Si `opcode==NEW_BETS`, construye `NewBets` y delega
       en `read_from(sock, length)`.
     - `read_from` en `NewBets` consume el contador de apuestas y, por cada una, usa `__read_bet` → `__read_pair` → `read_string` para reconstruir los pares `<k><v>`.
     - `read_string` valida longitud positiva, disponible en el `remaining` y decodifica UTF-8 con manejo de `UnicodeDecodeError` (conversión a `ProtocolError`).
     - Se mantiene un contador `remaining` consistente a lo largo de la lectura y se exige **`remaining == 0`** al final: cualquier desalineación dispara `ProtocolError`.

   - **Serialización (salida):**
     - Las respuestas usan `write_u8` + `write_i32(0)`. El método `write_struct` empaqueta con `struct.pack` y envía con `sock.sendall`, que garantiza la
       escritura de **todos los bytes** o un error, conforme al framing.

3. **Correcta separación de responsabilidades (dominio vs comunicación)**
   - La capa de **comunicación** valida el _wire format_, tipos, longitudes y opcodes; expone `NewBets.process()` pero sin mezclarla con E/S de sockets más allá
     del mensaje.
   - El **dominio** (`utils.Bet`, `store_bets`) se limita a representación y persistencia (CSV), sin conocer el protocolo binario.
   - La **aplicación** (`Server`) orquesta conexiones, maneja errores y decide qué respuesta enviar.

4. **Empleo correcto de sockets, manejo de errores y evitación de _short read_ / _short write_**
   - **Prevención de _short read_**:
     - `recv_exactly(sock, n)` implementa un bucle de lectura que acumula hasta leer **exactamente n bytes**, reintentando ante `InterruptedError`, propagando
       `timeout`/`OSError` como `ProtocolError`, y considerando `nrecv==0` como EOF (lanza `EOFError`).
     - Todas las lecturas de tipos de tamaño fijo (`read_struct`) pasan por `recv_exactly`, eliminando lecturas parciales.
     - El uso del contador `remaining` en `read_i32`/`read_string` fuerza la consistencia entre el `length` informado y el cuerpo efectivamente consumido.

   - **Prevención de _short write_**:
     - Las respuestas (`BETS_RECV_SUCCESS`/`BETS_RECV_FAIL`) se envían con `sock.sendall`, que bloquea hasta enviar el buffer completo o fallar, evitando _partial writes_.

   - **Manejo de errores y robustez**:
     - En `Server.__handle_client_connection`, se capturan `EOFError` y `ProtocolError`. En ambos casos se loguea el fallo, se intenta emitir `BETS_RECV_FAIL`
       (y si esa emisión falla, se loguea también) y **siempre** se cierra el socket en `finally`.
     - En éxito, se responde `BETS_RECV_SUCCESS` y luego se invoca `msg.process()`; véase el siguiente punto.

5. **Procesamiento de negocio y logging conforme a la consigna**
   - `NewBets.process()` delega en `utils.store_bets(self.bets)` la persistencia y, para **cada apuesta**, emite el log requerido por el enunciado:

     ```
     action: apuesta_almacenada | result: success | dni: %s | numero: %s
     ```

   - `Server.__handle_client_connection` registra la recepción del mensaje (`receive_message | success`) e implementa la semántica de _ack temprano_: responde
     `BETS_RECV_SUCCESS` al cliente (confirmación de recepción y parseo exitosos) y **luego** procesa y persiste. Esta elección desacopla la latencia del cliente
     de la E/S en disco. Si se quisiera confirmar _persistencia_ y no solo _recepción_, el protocolo podría evolucionar para que el ack llegue **después** del
     `process()` o para transportar un código de error; tal cambio no es requerido por la consigna actual.
   - En caso de errores de protocolo o EOF, se responde `BETS_RECV_FAIL` como indica la especificación.

6. **Señales, ciclo de vida y cierre ordenado**
   - `Server.run()` registra un handler de `SIGTERM` (`__stop_running`) que setea `_running=False` y **cierra el socket de escucha**. Así, si el proceso
     estaba bloqueado en `accept()`, este falla con `OSError`, se detecta que `_running=False` y el loop finaliza sin quedar colgado (graceful shutdown).
   - Al salir del loop se invoca `logging.shutdown()` para vaciar buffers y cerrar handlers de logging.

7. **Configuración y validaciones auxiliares**
   - `server/main.py` unifica configuración desde **variables de entorno** y/o `config.ini` (`SERVER_PORT`, `SERVER_LISTEN_BACKLOG`, `LOGGING_LEVEL`),
     con verificación y reporting temprano de errores de parseo (`KeyError`/`ValueError`).
   - `utils.Bet` valida/coerce tipos de dominio (e.g., `agency` y `number` a `int`, `birthdate` con `fromisoformat`), que protege contra entradas mal formadas
     aun cuando el framing fuera correcto.

---

**Detalles de diseño relevantes**

- **Validación estricta del body:** `NewBets.__read_bet` exige exactamente **6 pares** y la presencia de **todas** las claves requeridas; ante cualquier desvío,
  se lanza `ProtocolError("invalid body")`. Esto evita estados intermedios inconsistentes y previene escritura de registros corruptos.
- **Endianness y tamaños homogéneos:** todos los enteros del protocolo son `int32` LE; las strings usan prefijo de longitud `int32` y se validan
  (longitud positiva y bytes suficientes).
- **Manejo de Unicode:** los cuerpos de strings se decodifican en UTF-8; errores de decodificación se traducen a `ProtocolError("invalid body")`, manteniendo unívoca
  la semántica de error de protocolo.
