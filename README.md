# joss_smtp 2.0

`joss_smtp` envia correo mediante SMTP, STARTTLS o TLS implicito. Se entrega como JP v2 con bytecode, indice de IntelliSense y sidecars para Windows, Linux y macOS (amd64 y arm64). Una aplicacion Joss no necesita `use`, Go ni una instalacion local de un cliente SMTP.

## Instalacion

```bash
joss pub add joss_smtp 2.0.0
```

El cargador de plugins registra `SmtpClient` automaticamente al iniciar la aplicacion.

## Configuracion

Configura el servidor en `env.joss`:

| Variable | Uso |
| --- | --- |
| `MAIL_HOST` | Host SMTP. |
| `MAIL_PORT` | Puerto SMTP, por ejemplo `587` para STARTTLS o `465` para TLS. |
| `MAIL_USERNAME` | Usuario por defecto. |
| `MAIL_PASSWORD` | Contrasena o clave de aplicacion por defecto. |
| `MAIL_FROM_NAME` | Nombre visible del remitente. |
| `BREVO_API` | Compatibilidad con la configuracion de Brevo cuando corresponda. |

No guardes secretos dentro del codigo ni los incluyas en el paquete JP.

## Uso

```joss
$mail = new SmtpClient()
$ok = $mail->auth("user@example.com", "secret")
    ->secure(true)
    ->timeout(30)
    ->send("destino@example.com", "Asunto", "<b>Hola</b>")

($ok) ? {
    Console::log("Correo enviado")
} : {
    Console::log($mail->lastError())
}
```

`auth($user, $pass)`, `secure($enabled)` y `timeout($seconds)` son configuradores fluentes. `send($to, $subject, $body)` devuelve `true` solo si el proveedor confirma el envio. Si falla, devuelve `false` y `lastError()` contiene la causa; no se transforma un fallo en un `nil` silencioso.

## Distribucion y desarrollo

`joss_smtp.jp` contiene la API Joss y los binarios nativos para seis targets. `META-INF/joss-symbols.json` hace que la extension de VS Code muestre las firmas de `SmtpClient` sin analizar el codigo fuente del plugin.

Para reconstruir el paquete se necesita Go y Joss 3.6.0 o posterior. El script de distribucion central compila y valida los sidecars; el usuario final solo instala el JP publicado.
