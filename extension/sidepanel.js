// Persistent ChromeOS Renogy Bluetooth Low Energy Proxy

const RENOGY_SERVICE_WRITE = '0000ffd0-0000-1000-8000-00805f9b34fb';
const RENOGY_CHAR_WRITE = '0000ffd1-0000-1000-8000-00805f9b34fb';
const RENOGY_SERVICE_READ = '0000fff0-0000-1000-8000-00805f9b34fb';
const RENOGY_CHAR_NOTIFY = '0000fff1-0000-1000-8000-00805f9b34fb';

const ALL_SERVICES = [
  RENOGY_SERVICE_WRITE,
  RENOGY_SERVICE_READ,
  '0000ffe0-0000-1000-8000-00805f9b34fb',
  'battery_service',
  'device_information'
];

let ws = null;
let bleDevice = null;
let gattServer = null;
let writeChar = null;
let notifyChar = null;
let pollInterval = null;
let rxBuffer = new Uint8Array(0);
let isConnectingGatt = false;
let reconnectTimer = null;
let reconnectAttempts = 0;
let lastPacketTime = 0;

function log(msg, cls = '') {
  const box = document.getElementById('logBox');
  if (!box) return;
  const t = new Date().toLocaleTimeString();
  const d = document.createElement('div');
  if (cls) d.className = cls;
  d.textContent = `[${t}] ${msg}`;
  box.appendChild(d);
  box.scrollTop = box.scrollHeight;
}

function calcCrc16(buf) {
  let crc = 0xFFFF;
  for (let i = 0; i < buf.length; i++) {
    crc ^= buf[i];
    for (let j = 8; j !== 0; j--) {
      if ((crc & 1) !== 0) crc = (crc >> 1) ^ 0xA001;
      else crc >>= 1;
    }
  }
  return [crc & 0xFF, (crc >> 8) & 0xFF];
}

function initWebSocket() {
  ws = new WebSocket('ws://localhost:8765');
  ws.onopen = () => {
    const badge = document.getElementById('proxyBadge');
    if (badge) {
      badge.className = 'badge badge-online';
      badge.textContent = 'Bridge Online';
    }
    log('Connected to Linux WebSocket!', 'log-ok');
    autoConnectExisting();
  };
  ws.onclose = () => {
    const badge = document.getElementById('proxyBadge');
    if (badge) {
      badge.className = 'badge badge-offline';
      badge.textContent = 'Bridge Offline';
    }
    setTimeout(initWebSocket, 3000);
  };
}

function sendToLinux(obj) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(obj));
  }
}

async function autoConnectExisting() {
  if (isConnectingGatt || (bleDevice && bleDevice.gatt && bleDevice.gatt.connected)) return;
  if (!navigator.bluetooth || !navigator.bluetooth.getDevices) return;
  try {
    const devices = await navigator.bluetooth.getDevices();
    if (devices.length > 0) {
      const renogy = devices.find(d => d.name && (d.name.includes('BT') || d.name.includes('RN') || d.name.includes('Rover') || d.name.includes('Solar'))) || devices[0];
      log(`Found authorized controller: ${renogy.name || 'Renogy BT'}`, 'log-ok');
      await connectToDevice(renogy);
    }
  } catch (e) {
    log(`Auto-connect note: ${e.message}`);
  }
}

async function requestPairing() {
  try {
    log('Scanning for Renogy BT-1 / BT-2...');
    bleDevice = await navigator.bluetooth.requestDevice({
      filters: [{ services: [RENOGY_SERVICE_WRITE] }],
      optionalServices: ALL_SERVICES
    });
    await connectToDevice(bleDevice);
  } catch (err) {
    log(`Pairing error: ${err.message}`, 'log-err');
  }
}

async function connectToDevice(dev) {
  if (!dev || isConnectingGatt) return;
  isConnectingGatt = true;
  bleDevice = dev;
  const name = dev.name || 'Renogy BT Module';
  const devNameEl = document.getElementById('devName');
  if (devNameEl) devNameEl.textContent = name;
  log(`Connecting GATT session to ${name}...`);

  dev.removeEventListener('gattserverdisconnected', onDisconnected);
  dev.addEventListener('gattserverdisconnected', onDisconnected);

  try {
    gattServer = await dev.gatt.connect();
    reconnectAttempts = 0;
    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }

    const bleStatus = document.getElementById('bleStatus');
    if (bleStatus) {
      bleStatus.className = 'badge badge-solar';
      bleStatus.textContent = 'Connected';
    }
    log(`GATT Connected to ${name}! Hooking channels...`, 'log-ok');

    const rxService = await gattServer.getPrimaryService(RENOGY_SERVICE_READ);
    notifyChar = await rxService.getCharacteristic(RENOGY_CHAR_NOTIFY);
    await notifyChar.startNotifications();
    notifyChar.removeEventListener('characteristicvaluechanged', onDataChunk);
    notifyChar.addEventListener('characteristicvaluechanged', onDataChunk);

    const txService = await gattServer.getPrimaryService(RENOGY_SERVICE_WRITE);
    writeChar = await txService.getCharacteristic(RENOGY_CHAR_WRITE);

    sendToLinux({ type: 'gatt_connected', name: name, id: dev.id });

    // Start auto polling (5 seconds)
    if (pollInterval) clearInterval(pollInterval);
    pollInterval = setInterval(pollTelemetry, 5000);
    pollTelemetry();

  } catch (e) {
    log(`GATT connection failed: ${e.message}`, 'log-err');
    scheduleReconnect();
  } finally {
    isConnectingGatt = false;
  }
}

async function pollTelemetry() {
  if (!writeChar || !gattServer || !gattServer.connected) return;
  try {
    const raw = new Uint8Array([0xFF, 0x03, 0x01, 0x00, 0x00, 0x22]);
    const crc = calcCrc16(raw);
    const frame = new Uint8Array([...raw, ...crc]);
    if (writeChar.properties.write) {
      await writeChar.writeValueWithResponse(frame);
    } else {
      await writeChar.writeValueWithoutResponse(frame);
    }
  } catch (e) {
    // Drop
  }
}

function onDataChunk(event) {
  lastPacketTime = Date.now();
  const chunk = new Uint8Array(event.target.value.buffer, event.target.value.byteOffset, event.target.value.byteLength);
  sendToLinux({ type: 'notification', characteristic: RENOGY_CHAR_NOTIFY, bytes: Array.from(chunk) });

  const newBuf = new Uint8Array(rxBuffer.length + chunk.length);
  newBuf.set(rxBuffer);
  newBuf.set(chunk, rxBuffer.length);
  rxBuffer = newBuf;

  if (rxBuffer.length >= 3 && rxBuffer[1] === 0x03) {
    const expected = 3 + rxBuffer[2] + 2;
    if (rxBuffer.length >= expected) {
      const full = rxBuffer.slice(0, expected);
      rxBuffer = rxBuffer.slice(expected);
      parseFrame(full);
    }
  }
}

function parseFrame(bytes) {
  if (bytes.length < 5 || bytes[1] !== 0x03) return;
  const data = bytes.slice(3, 3 + bytes[2]);

  if (bytes[2] >= 60 || data.length >= 20) {
    const soc = (data[0] << 8) | data[1];
    const volts = ((data[2] << 8) | data[3]) * 0.1;
    const pvWatts = (data[18] << 8) | data[19];
    const stateCode = data.length >= 66 ? data[65] : 0;
    const stateMap = { 0: 'Deactivated', 1: 'Active', 2: 'MPPT', 3: 'Equalize', 4: 'Boost', 5: 'Float', 6: 'Limit' };

    const pvWattsEl = document.getElementById('pvWatts');
    if (pvWattsEl) pvWattsEl.textContent = `${pvWatts} W`;
    const battSocEl = document.getElementById('battSoc');
    if (battSocEl) battSocEl.textContent = `${soc} %`;
    const battVoltsEl = document.getElementById('battVolts');
    if (battVoltsEl) battVoltsEl.textContent = `${volts.toFixed(1)} V`;
    const chargeStateEl = document.getElementById('chargeState');
    if (chargeStateEl) chargeStateEl.textContent = stateMap[stateCode] || `Code ${stateCode}`;

    log(`Solar: ${pvWatts}W | Batt: ${volts.toFixed(1)}V (${soc}%)`, 'log-solar');
  }
}

function onDisconnected() {
  log('Bluetooth disconnected. Scheduling automatic reconnection...', 'log-err');
  const bleStatus = document.getElementById('bleStatus');
  if (bleStatus) {
    bleStatus.className = 'badge badge-offline';
    bleStatus.textContent = 'Reconnecting...';
  }
  if (pollInterval) {
    clearInterval(pollInterval);
    pollInterval = null;
  }
  writeChar = null;
  notifyChar = null;
  sendToLinux({ type: 'disconnected', name: (bleDevice && bleDevice.name) || 'Renogy Module' });
  scheduleReconnect();
}

function scheduleReconnect() {
  if (isConnectingGatt) return;
  reconnectAttempts++;
  const delayMs = Math.min(2000 * Math.pow(1.5, reconnectAttempts - 1), 15000);
  log(`Auto-reconnect attempt #${reconnectAttempts} in ${(delayMs/1000).toFixed(1)}s...`, 'log-solar');

  if (reconnectTimer) clearTimeout(reconnectTimer);
  reconnectTimer = setTimeout(async () => {
    reconnectTimer = null;
    if (bleDevice) {
      await connectToDevice(bleDevice);
    } else {
      await autoConnectExisting();
    }
  }, delayMs);
}

// Watchdog for stalled GATT channels
setInterval(() => {
  if (bleDevice && bleDevice.gatt && bleDevice.gatt.connected && lastPacketTime > 0) {
    if (Date.now() - lastPacketTime > 15000 && !isConnectingGatt) {
      log('Watchdog: Telemetry stalled (>15s). Resetting GATT...', 'log-err');
      try { bleDevice.gatt.disconnect(); } catch (e) {}
      onDisconnected();
    }
  } else if (bleDevice && (!bleDevice.gatt || !bleDevice.gatt.connected) && !isConnectingGatt && !reconnectTimer) {
    scheduleReconnect();
  }
}, 4000);

const pairBtn = document.getElementById('pairBtn');
if (pairBtn) pairBtn.addEventListener('click', requestPairing);

window.addEventListener('load', () => {
  initWebSocket();
});
