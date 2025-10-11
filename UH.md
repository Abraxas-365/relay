# FactúraMelo - Descripción del Proyecto

**FactúraMelo** es una **plataforma integral de automatización y protección de gestión de facturas** diseñada para empresas en **Latinoamérica y Europa**. El sistema se integra directamente con los sistemas tributarios locales (SUNAT en Perú, SAT en México, SII en Chile, AFIP en Argentina, etc.) proporcionando una solución completa para el procesamiento, validación y gestión de facturas electrónicas.

## 🎯 Objetivo Principal

Automatizar completamente el ciclo de vida de la gestión contable y **proteger a las empresas contra fraudes** comunes en la facturación electrónica, incluyendo:

### **Protección Contra Fraude**
- **Fraude por "silencio administrativo"**: En muchos países de LATAM, si no rechazas una factura dentro de un plazo legal (8 días en Perú, varía por país), se considera automáticamente aceptada. Los estafadores emiten facturas falsas y esperan que pase el plazo.
- **Doble cobro**: Proveedores que presentan la misma factura múltiples veces o la venden a factoring y luego la cobran directamente.
- **Facturas fantasma**: Facturas emitidas en el sistema tributario pero nunca notificadas al comprador.
- **Servicios no prestados**: Facturas por mercadería nunca entregada o servicios nunca realizados.

### **Automatización Completa**
- **Descarga automática** desde sistemas tributarios oficiales
- **Validación en tiempo real** con autoridades fiscales
- **Notificaciones inteligentes** a las áreas correspondientes
- **Gestión de proveedores confiables**
- **Flujos de aprobación automatizados**
- **Cumplimiento tributario** automático por país

## 🏗️ Arquitectura del Sistema por Dominios

El sistema está construido siguiendo **Domain-Driven Design** con los siguientes dominios:

### **🔐 IAM Domain** (Gestión de Identidad y Acceso)

#### **Users (Usuarios)**
- Registro y autenticación OAuth (Google Workspace, Microsoft 365, Auth0)
- Gestión de perfiles y preferencias
- Sistema de invitaciones por email
- Estados de usuario (pendiente, activo, inactivo, suspendido)
- Tracking de sesiones y auditoría de accesos

#### **Tenants (Empresas/Multi-tenancy)**
- Aislamiento completo de datos por empresa
- Validación de números fiscales (RUC, RFC, NIT, CUIT, etc.) según país
- Gestión de suscripciones y planes
- Límites por plan (usuarios, facturas/mes, integraciones)
- Configuraciones empresariales (branding, preferencias)

#### **Auth (Autenticación)**
- Gestión de tokens JWT con claims personalizados
- Refresh tokens con rotación automática
- Sesiones concurrentes y auto-logout por inactividad
- OAuth2 flow completo

#### **Roles (Control de Acceso)**
- **Roles predefinidos**: Admin Empresa, Contador, Compras, Aprobador por Área, Solo Lectura
- **Permisos granulares**: invoice.read, invoice.approve, supplier.manage, etc.
- **Roles personalizados** por empresa

---

### **📄 Invoices Domain** (Gestión de Facturas)

#### **Invoices (Facturas)**
- **Recepción automática**: Descarga directa desde sistemas tributarios cada hora
- **Recepción manual**: Upload de PDF/XML, recepción por email
- **OCR**: Digitalización de facturas físicas escaneadas
- **Validación automática**:
  - Validación fiscal en tiempo real
  - Verificación de número fiscal del proveedor
  - Detección de duplicados (número + proveedor + monto)
  - Validación de formato según estándar del país
- **Three-Way Matching**: Factura ↔ Orden de Compra ↔ Guía de Remisión
- **Estados del ciclo de vida**:
  - `PENDING_VENDOR_VERIFICATION` (recién recibida)
  - `PENDING_APPROVAL` (validada, esperando aprobación)
  - `APPROVED` (aprobada para pago)
  - `REJECTED` (rechazada con motivo)
  - `IN_DISPUTE` (en disputa con proveedor)
  - `CONTABILIZED` (contabilizada en el sistema)
- **Motor de reglas inteligente**:
  - Auto-aprobación para proveedores confiables bajo cierto monto
  - Ruteo automático según centro de costo/área
  - Escalamiento por montos (>$10,000 → Gerencia)
  - Alertas por patrones inusuales

#### **Attachments (Archivos Adjuntos)**
- Tipos soportados: PDF, XML (UBL/estándar local), Guías de Remisión, Órdenes de Compra
- Validación de tipos MIME y límites de tamaño
- Antivirus scanning automático
- Extracción automática de datos (parser XML, OCR)
- Storage seguro en cloud con URLs firmadas

---

### **📦 Purchase Orders Domain** (Órdenes de Compra)

#### **Purchase Orders (Órdenes de Compra)**
- Generación automática de números de orden
- Templates para órdenes recurrentes
- **Estados**: DRAFT, PENDING_APPROVAL, APPROVED, PARTIALLY_RECEIVED, COMPLETED, CANCELED
- Workflow de aprobación configurable
- Integración con proveedores (envío automático por email)
- Portal para confirmación de proveedores

#### **Items (Líneas de Orden)**
- Catálogo de productos/servicios maestro
- Gestión detallada de cantidades, precios y unidades
- Control de recepciones y entregas parciales
- Validación automática contra facturas

---

### **🏢 Suppliers/Vendors Domain** (Proveedores)

#### **Suppliers (Proveedores)**
- **Onboarding**: Registro con validación automática de número fiscal
- **Sistema de confiabilidad**:
  - **Proveedores de Confianza**: Auto-aprobación hasta cierto monto
  - **Nuevos Proveedores**: Aprobación manual obligatoria
  - **Lista Negra**: Proveedores bloqueados
  - Score dinámico basado en historial
- **Estados**: PENDING_VERIFICATION, ACTIVE, SUSPENDED, BLACKLISTED
- **Gestión comercial**: Términos de pago, límites de crédito, descuentos
- **Métricas de performance**: Tiempo de entrega, cumplimiento, calidad

#### **Contacts (Contactos)**
- Tipos: PRIMARY, ACCOUNTING, PURCHASES, TECHNICAL, LEGAL, COMMERCIAL
- Múltiples contactos por proveedor
- Comunicación automatizada por tipo de contacto
- Validación de información de contacto

#### **Supplier Portal (Portal de Proveedores)**
- Dashboard para proveedores con vista de órdenes pendientes
- Confirmación/rechazo de órdenes de compra
- Upload directo de facturas
- Tracking de estado de procesamiento
- Mensajería directa con áreas de compras

---

### **🏛️ Areas Domain** (Estructura Organizacional)

- Estructura jerárquica de departamentos y subdepartamentos
- Centros de costo y presupuestos por área
- Ruteo inteligente de facturas según reglas
- Gestión presupuestaria con alertas de sobregiros
- Métricas de eficiencia por área

---

### **⚙️ Workflows Domain** (Flujos de Trabajo)

- **Tipos de workflow**:
  - `SIMPLE`: Aprobación de un solo nivel
  - `MULTI_LEVEL`: Aprobación secuencial multinivel
  - `CONDITIONAL`: Basado en condiciones (monto, proveedor, etc.)
  - `PARALLEL`: Múltiples aprobadores en paralelo
- **Workflows predefinidos**:
  - Aprobación de facturas por monto y área
  - Aprobación de órdenes de compra por jerarquía
  - Onboarding de proveedores
  - Resolución de discrepancias
- Escalamiento automático por timeout
- Notificaciones inteligentes (Email/SMS/Push)

---

### **🔌 Integrations Domain** (Integraciones)

#### **Sistemas Tributarios por País**
- **Perú**: SUNAT/SIRE
- **México**: SAT
- **Chile**: SII
- **Colombia**: DIAN
- **Argentina**: AFIP
- **España**: AEAT
- **Y más países de LATAM y Europa**

**Funcionalidades**:
- OAuth2 con credenciales fiscales por empresa
- Validación en tiempo real
- Descarga automática de facturas electrónicas
- Gestión de períodos tributarios
- Reportes de validación y cumplimiento
- Actualización de crédito fiscal

#### **ERPs y Sistemas Contables**
- Conectores nativos: SAP, Oracle, Exact, Microsoft Dynamics
- APIs de sincronización bidireccional
- Mapeo de campos personalizable

#### **Bancos y Medios de Pago**
- Integración con principales bancos de cada país
- Detección automática de pagos
- Conciliación bancaria automatizada

#### **Servicios de Email**
- Procesamiento de facturas recibidas por email
- Envío masivo de comunicaciones
- Templates responsive para notificaciones

---

## 💰 Planes de Suscripción

| Plan | Precio | Facturas/mes | Usuarios | Características Principales |
|------|---------|--------------|----------|----------------------------|
| **Básico** | $0 | 15 | 1 | Aprobación simple, base de conocimiento |
| **Profesional** | $29 | 150 | 3 | Hasta 5 áreas, soporte email |
| **Negocios** | $149 | 400 | 10 | Aprobación multinivel, hasta 15 áreas, API básica, chat prioritario |
| **Corporativo** | $450 | 1,000 | 25 | Validación 3-vías, OCR, IA para digitalización, API avanzada, soporte telefónico |
| **Enterprise** | Desde $999 | 2,000+ | Ilimitados | IA avanzada (insights y detección de anomalías), conector nativo ERP, áreas sincronizadas desde ERP, gerente de cuenta dedicado |

---

## 🔑 Ventajas Clave del Sistema

### **1. Cobertura 100% Automática**
Como todas las facturas son electrónicas, FactúraMelo las obtiene **DIRECTAMENTE** del sistema tributario oficial:
- ✅ Capturas TODAS las facturas emitidas a tu número fiscal
- ✅ Detección instantánea cuando alguien emite una factura
- ✅ Sin depender de que el proveedor te las envíe
- ✅ Imposible que una factura pase desapercibida

### **2. Protección Contra Fraude por "Silencio"**
- 🚨 Detecta facturas antes que tú: Sistema las ve en el portal tributario y alerta de inmediato
- ⏱️ Contador automático del plazo legal por país
- 📱 Notificaciones urgentes al responsable
- 📋 Validación automática:
  - ¿Tienes Orden de Compra? → NO = 🚨 POSIBLE FRAUDE
  - ¿Recibiste mercadería? → NO = 🚨 POSIBLE FRAUDE
  - ¿Conoces este gasto? → NO = 🚨 ALERTA INMEDIATA
- ❌ Rechazo con un click: Genera carta formal y registra en sistema tributario
- 🔐 Nunca te sorprenden con facturas desconocidas

### **3. Eliminación de Dobles Pagos**
- ✅ Cross-check automático con autoridades fiscales
- 🚫 Detecta duplicados: Mismo número + proveedor = BLOQUEO AUTOMÁTICO
- 📦 Three-way matching: Factura ↔ Orden de Compra ↔ Guía de Remisión
- 🔍 Verifica estado del proveedor en tiempo real

### **4. Historial Completo Descargado Automáticamente**
- 📚 TODAS tus facturas desde el sistema tributario
- 🔍 Búsqueda instantánea por fecha, proveedor, monto, estado
- 💾 Archivos originales XML + PDF (100% auditable)
- 📊 Reportes fiscales generados automáticamente
- ⏰ Timeline completo con trazabilidad total

### **5. Aprobación Inteligente por Áreas**
- 📍 Ruteo automático al área/departamento correcto
- 📱 Notificación inmediata con plazo de acción
- 🔍 Validaciones inteligentes pre-aprobación
- ⏰ Escalamiento automático si no hay respuesta
- 📝 Trazabilidad completa de todas las acciones

### **6. Integración con ERPs**
- 🔗 Exporta facturas validadas a SAP, Oracle, Exact, Dynamics, etc.
- ⚡ Sincronización automática bidireccional
- 🔌 APIs REST para conectar cualquier sistema

---

## 🚀 Flujo Automatizado Completo

```
🏛️ Sistema Tributario Local (SUNAT/SAT/SII/AFIP/AEAT)
   ↓ (Sincronización cada hora)
🤖 FactúraMelo descarga TODAS las facturas nuevas con tu número fiscal
   ↓
⏰ Sistema inicia contador automático (plazo legal del país)
   ↓
🔍 Validación instantánea:
   ✅ ¿Existe en el sistema tributario? (siempre sí)
   ❌ ¿Ya la pagaste? (detector de duplicados)
   ❌ ¿Hay Orden de Compra? → NO = 🚨 ALERTA
   ❌ ¿Hay Guía de Remisión/Entrega? → NO = 🚨 ALERTA
   ✅ ¿Proveedor de confianza? → Score de riesgo
   ↓
📱 Notificación INMEDIATA al área responsable
   "⚠️ Nueva factura de [Proveedor] - $[Monto] - Quedan X días"
   ↓
👤 Responsable revisa en el dashboard:
   • Factura original (PDF/XML del sistema tributario)
   • Orden de Compra (si existe)
   • Guía de Remisión (si existe)
   • Historial del proveedor (score, facturas previas)
   • Impacto presupuestal del área
   ↓
✅ APROBAR → Pasa automáticamente a cola de pago
   O
❌ RECHAZAR → Sistema genera carta formal + registra en sistema tributario
   ↓
⏰ Si NO responde en plazo configurado:
   • Día X-2 → 🚨 Alerta naranja
   • Día X-1 → 🚨🚨 Alerta roja + escalamiento a gerencia
   ↓
💾 TODO queda registrado con trazabilidad completa (auditable)
   ↓
📊 Se exporta automáticamente a ERP/Sistema Contable
```

---

## 💡 Comparativa: Antes vs Con FactúraMelo

| Situación | ❌ Sin FactúraMelo | ✅ Con FactúraMelo |
|-----------|-------------------|-------------------|
| **Cobertura** | Dependes de que te envíen las facturas | Obtienes TODAS directo del sistema tributario |
| **Facturas perdidas** | Quedan en bandeja de spam o email personal | Imposible perder una factura |
| **Conocimiento** | No sabes qué facturas te emitieron | Alertas instantáneas de todas las emisiones |
| **Plazo legal** | Olvidas fechas límite fácilmente | Contador automático + alertas progresivas |
| **Fraude** | Te cobran por facturas desconocidas | Detectas y rechazas antes del plazo |
| **Validación** | Trabajo manual horas/días | Validación automática en segundos 24/7 |
| **Dobles pagos** | Pagas facturas duplicadas sin darte cuenta | Detector automático de duplicados |
| **Historial** | Archivos dispersos en emails y carpetas | Historial completo descargado del sistema tributario |
| **Aprobaciones** | Procesos manuales con email | Flujos automatizados con trazabilidad |
| **Presupuesto** | Control manual de gastos por área | Alertas automáticas de sobregiros |

---

## 🎯 Casos Reales de Fraude Evitados

### **Caso 1: Fraude del Silencio (Evitado en Tiempo Real)**

**❌ Antes:**
1. Lunes 9:00am: Proveedor emite factura falsa por $20,000 en sistema tributario
2. Nunca te avisa - espera que pase el plazo legal
3. La factura está registrada oficialmente pero no lo sabes
4. Pasan 8 días → Se acepta automáticamente por ley
5. Banco/factoring te cobra → Pierdes $20,000 😱

**✅ Con FactúraMelo:**
1. Lunes 9:00am: Proveedor emite factura en sistema tributario
2. Lunes 10:00am: FactúraMelo la detecta automáticamente 🚨
3. Lunes 10:05am: Alerta al responsable: "⚠️ Factura sin Orden de Compra"
4. Lunes 11:00am: Responsable revisa → No reconoce el gasto
5. Lunes 11:30am: RECHAZA con un click
6. Sistema genera carta formal y registra rechazo en autoridad fiscal
7. **Resultado**: $20,000 protegidos 🎯

---

### **Caso 2: Factura "Fantasma"**

**❌ Antes:**
1. Proveedor emite factura por "consultoría" nunca prestada
2. No la envía por ningún lado
3. Espera silenciosamente los días legales
4. Tú ni sabes que existe
5. Se acepta automáticamente → Te cobran

**✅ Con FactúraMelo:**
1. Proveedor emite en sistema tributario
2. Sistema la detecta en minutos 🤖
3. 🚨 "Nueva factura sin Orden de Compra - Acción requerida"
4. Área revisa → No reconoce el servicio
5. RECHAZO inmediato antes del plazo
6. **Fraude evitado** 🎯

---

### **Caso 3: Doble Cobro**

**❌ Antes:**
1. Pagas factura de $15,000
2. Semanas después, proveedor la presenta otra vez
3. Área diferente la aprueba (no sabe que ya se pagó)
4. Pagas $30,000 por $15,000

**✅ Con FactúraMelo:**
1. Primera vez: Factura se procesa y paga normalmente
2. Intenta presentarla otra vez → 🚨 "DUPLICADO DETECTADO"
3. Sistema muestra: "Ya pagado el 15/03/2025 - $15,000"
4. Automáticamente bloqueado 🎯
5. **$15,000 ahorrados**

---

## 📊 ROI y Ahorros

### **Ahorro Promedio Anual**
- 💰 **$50,000 - $200,000** en fraudes evitados
- ⏱️ **+80 horas/mes** ahorradas en procesamiento manual
- 📉 **2.5%** reducción en desembolsos por errores o duplicados
- 🎯 **+35 mil millones USD** perdidos anualmente en LATAM por mal manejo de CxP (según McKinsey)

### **Tiempo de Implementación**
- ⚡ Setup inicial: 2-5 días
- 🔗 Integración con ERP: 1-2 semanas
- 📚 Capacitación de usuarios: 1 día
- 🚀 ROI positivo: Primer mes

---

## 🌍 Cobertura Geográfica

### **Latinoamérica**
- 🇵🇪 Perú (SUNAT/SIRE)
- 🇲🇽 México (SAT)
- 🇨🇱 Chile (SII)
- 🇨🇴 Colombia (DIAN)
- 🇦🇷 Argentina (AFIP)
- 🇧🇷 Brasil (SEFAZ)
- 🇪🇨 Ecuador (SRI)
- Y más países...

### **Europa**
- 🇪🇸 España (AEAT)
- 🇵🇹 Portugal (AT)
- 🇮🇹 Italia (AdE)
- 🇫🇷 Francia (DGFiP)
- Y más países...

Cada país con su integración específica al sistema tributario local y cumplimiento de normativas fiscales.

---

## 🔐 Seguridad y Cumplimiento

- 🔒 Cifrado end-to-end
- 🏛️ Cumplimiento normativo por país
- 📋 Auditoría completa de todas las acciones
- 💾 Backup automático y redundancia
- 🛡️ Protección contra ataques y antivirus integrado
- 🔑 Autenticación multi-factor (MFA)

---

## 🎯 En Resumen

**FactúraMelo** es la solución definitiva para empresas de LATAM y Europa que buscan:

✅ **Protección total contra fraude** de facturas falsas
✅ **Automatización completa** del ciclo de vida contable
✅ **Cumplimiento fiscal** garantizado en cada país
✅ **Integración perfecta** con ERPs y sistemas existentes
✅ **Ahorro significativo** en tiempo y dinero
✅ **Tranquilidad** de tener control total sobre tus facturas

**Ventaja Definitiva**: *"Si está en el sistema tributario oficial, lo sabemos de inmediato y actuamos antes del plazo legal"* 🎯
