// Chrome Extension Service Worker - Renogy Solar BLE Bridge

chrome.runtime.onInstalled.addListener(() => {
  console.log('[Renogy BLE Bridge] Extension installed.');
  chrome.sidePanel.setPanelBehavior({ openPanelOnActionClick: true });
});

chrome.action.onClicked.addListener(async (tab) => {
  // Opens the side panel on extension icon click
  await chrome.sidePanel.open({ tabId: tab.id });
});

// Periodic keep-alive alarm
chrome.alarms.create('keepAlive', { periodInMinutes: 1 });
chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === 'keepAlive') {
    // Service worker active check
  }
});
