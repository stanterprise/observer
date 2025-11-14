#!/usr/bin/env node

/**
 * Simple WebSocket test client for Observer
 * 
 * Usage: node websocket-test.js [ws-url]
 */

const WebSocket = require('ws');

const WS_URL = process.argv[2] || 'ws://localhost:8080/ws';

console.log(`Connecting to ${WS_URL}...`);

const ws = new WebSocket(WS_URL);

ws.on('open', () => {
  console.log('✅ Connected to WebSocket');
  console.log('Waiting for events...\n');
});

ws.on('message', (data) => {
  try {
    const event = JSON.parse(data.toString());
    console.log('📨 Event received:');
    console.log('  Type:', event.type);
    console.log('  Time:', new Date(event.timestamp).toISOString());
    console.log('  Data:', JSON.stringify(event.data, null, 2));
    console.log('');
  } catch (e) {
    console.error('❌ Failed to parse event:', e.message);
    console.error('Raw data:', data.toString());
  }
});

ws.on('error', (error) => {
  console.error('❌ WebSocket error:', error.message);
});

ws.on('close', () => {
  console.log('👋 Connection closed');
  process.exit(0);
});

// Handle Ctrl+C gracefully
process.on('SIGINT', () => {
  console.log('\nClosing connection...');
  ws.close();
});

console.log('Press Ctrl+C to disconnect');
