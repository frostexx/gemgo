import React, { useState, useEffect } from 'react';
import axios from 'axios';
import './App.css';

function App() {
  const [currentView, setCurrentView] = useState('login');
  const [walletData, setWalletData] = useState(null);
  const [seedPhrase, setSeedPhrase] = useState('');
  const [sponsorSeedPhrase, setSponsorSeedPhrase] = useState('');
  const [withdrawalAddress, setWithdrawalAddress] = useState('');
  const [selectedBalance, setSelectedBalance] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [messages, setMessages] = useState([]);
  const [ws, setWs] = useState(null);

  const addMessage = (message, type = 'info') => {
    setMessages(prev => [...prev, { 
      id: Date.now(), 
      text: message, 
      type, 
      timestamp: new Date().toLocaleTimeString() 
    }]);
  };

  const handleLogin = async () => {
    setIsLoading(true);
    try {
      const response = await axios.post('/api/login', {
        seed_phrase: seedPhrase,
        sponsor_seed_phrase: sponsorSeedPhrase
      });
      
      setWalletData(response.data);
      setCurrentView('dashboard');
      addMessage('✅ Login successful!', 'success');
    } catch (error) {
      addMessage('❌ Login failed: ' + (error.response?.data?.message || error.message), 'error');
    } finally {
      setIsLoading(false);
    }
  };

  const handleWithdraw = () => {
    if (!selectedBalance || !withdrawalAddress) {
      addMessage('❌ Please select a locked balance and enter withdrawal address', 'error');
      return;
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws/withdraw`;
    const websocket = new WebSocket(wsUrl);

    websocket.onopen = () => {
      addMessage('🔌 Connected to withdrawal service', 'info');
      const withdrawData = {
        seed_phrase: seedPhrase,
        sponsor_seed_phrase: sponsorSeedPhrase,
        locked_balance_id: selectedBalance,
        withdrawal_address: withdrawalAddress,
        amount: "0"
      };
      websocket.send(JSON.stringify(withdrawData));
    };

    websocket.onmessage = (event) => {
      const response = JSON.parse(event.data);
      handleWebSocketResponse(response);
    };

    websocket.onerror = (error) => {
      addMessage('❌ WebSocket error: ' + error.message, 'error');
    };

    websocket.onclose = () => {
      addMessage('🔌 Connection closed', 'info');
    };

    setWs(websocket);
  };

  const handleWebSocketResponse = (response) => {
    switch(response.action) {
      case 'sponsor_validated':
        addMessage('✅ Sponsor wallet validated successfully', 'success');
        break;
      case 'available_balance_check':
        addMessage('💰 ' + response.message, response.success ? 'success' : 'warning');
        break;
      case 'available_withdrawn':
        addMessage(`✅ Available balance withdrawn: ${response.amount} PI`, 'success');
        break;
      case 'scheduled':
        addMessage('📅 ' + response.message, 'success');
        break;
      case 'countdown':
        addMessage('⏱️ ' + response.message, 'info');
        break;
      case 'executing':
        addMessage('🚀 ' + response.message, 'success');
        break;
      case 'completed':
        addMessage(response.success ? '🎉 ' + response.message : '❌ ' + response.message, 
                  response.success ? 'success' : 'error');
        break;
      case 'warning':
        addMessage('⚠️ ' + response.message, 'warning');
        break;
      default:
        addMessage(response.message || 'Unknown response', response.success ? 'success' : 'error');
    }
  };

  useEffect(() => {
    return () => {
      if (ws) {
        ws.close();
      }
    };
  }, [ws]);

  if (currentView === 'login') {
    return (
      <div className="min-h-screen bg-gradient-to-br from-blue-900 to-purple-900 flex items-center justify-center p-4">
        <div className="bg-white rounded-xl shadow-2xl p-8 w-full max-w-md">
          <div className="text-center mb-8">
            <h1 className="text-3xl font-bold text-gray-800 mb-2">PI Wallet BOT</h1>
            <p className="text-gray-600">Enhanced Competitive Edition</p>
          </div>

          <div className="space-y-6">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Wallet Seed Phrase *
              </label>
              <textarea
                value={seedPhrase}
                onChange={(e) => setSeedPhrase(e.target.value)}
                placeholder="Enter your wallet seed phrase..."
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 h-20 resize-none"
                required
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Sponsor Seed Phrase (Optional)
              </label>
              <textarea
                value={sponsorSeedPhrase}
                onChange={(e) => setSponsorSeedPhrase(e.target.value)}
                placeholder="Enter sponsor wallet seed phrase for claiming fees..."
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-green-500 h-20 resize-none"
              />
              <p className="text-xs text-gray-500 mt-1">
                💡 Sponsor pays claiming fees, main wallet pays transfer fees
              </p>
            </div>

            <button
              onClick={handleLogin}
              disabled={!seedPhrase || isLoading}
              className="w-full bg-blue-600 text-white py-3 rounded-md hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed font-medium transition-colors"
            >
              {isLoading ? '🔄 Logging in...' : '🚀 Login'}
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-900 to-purple-900 p-4">
      <div className="max-w-6xl mx-auto">
        <div className="bg-white rounded-xl shadow-2xl p-6 mb-6">
          <div className="flex justify-between items-center mb-6">
            <h1 className="text-2xl font-bold text-gray-800">PI Wallet Dashboard</h1>
            <button
              onClick={() => {setCurrentView('login'); setWalletData(null); setMessages([]);}}
              className="bg-red-500 text-white px-4 py-2 rounded-md hover:bg-red-600"
            >
              Logout
            </button>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
            <div className="bg-green-50 p-4 rounded-lg">
              <h3 className="font-semibold text-green-800">Available Balance</h3>
              <p className="text-2xl font-bold text-green-600">{walletData?.available_balance} PI</p>
            </div>
            <div className="bg-blue-50 p-4 rounded-lg">
              <h3 className="font-semibold text-blue-800">Wallet Address</h3>
              <p className="text-sm text-blue-600 break-all">{walletData?.wallet_address}</p>
            </div>
            <div className="bg-purple-50 p-4 rounded-lg">
              <h3 className="font-semibold text-purple-800">Sponsor Status</h3>
              <p className="text-sm text-purple-600">
                {walletData?.sponsor_address ? `✅ Active: ${walletData.sponsor_balance} PI` : '❌ Not Set'}
              </p>
            </div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="bg-gray-50 p-4 rounded-lg">
              <h3 className="font-semibold text-gray-800 mb-4">🔒 Locked Balances</h3>
              <div className="space-y-2 max-h-60 overflow-y-auto">
                {walletData?.locked_balances?.map((balance, index) => (
                  <div key={index} className="bg-white p-3 rounded border">
                    <div className="flex justify-between items-center">
                      <span className="font-medium">{balance.amount} PI</span>
                      <button
                        onClick={() => setSelectedBalance(balance.id)}
                        className={`px-3 py-1 rounded text-sm ${
                          selectedBalance === balance.id 
                            ? 'bg-blue-500 text-white' 
                            : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
                        }`}
                      >
                        {selectedBalance === balance.id ? 'Selected' : 'Select'}
                      </button>
                    </div>
                    <p className="text-xs text-gray-500">ID: {balance.id}</p>
                  </div>
                ))}
              </div>
            </div>

            <div className="bg-gray-50 p-4 rounded-lg">
              <h3 className="font-semibold text-gray-800 mb-4">🚀 Withdrawal Setup</h3>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Withdrawal Address
                  </label>
                  <input
                    type="text"
                    value={withdrawalAddress}
                    onChange={(e) => setWithdrawalAddress(e.target.value)}
                    placeholder="Enter destination wallet address..."
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
                
                <button
                  onClick={handleWithdraw}
                  disabled={!selectedBalance || !withdrawalAddress}
                  className="w-full bg-green-600 text-white py-3 rounded-md hover:bg-green-700 disabled:bg-gray-400 disabled:cursor-not-allowed font-medium"
                >
                  🎯 Start Competitive Withdrawal
                </button>
                
                <div className="text-xs text-gray-600 space-y-1">
                  <p>⚡ Uses competitive fees (3.2-9.4 PI)</p>
                  <p>🏃‍♂️ Concurrent claiming & transfer</p>
                  <p>🌊 Network flooding protection</p>
                  <p>🎯 100ms head start advantage</p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="bg-white rounded-xl shadow-2xl p-6">
          <h3 className="font-semibold text-gray-800 mb-4">📊 Activity Log</h3>
          <div className="bg-black text-green-400 p-4 rounded-lg h-64 overflow-y-auto font-mono text-sm">
            {messages.map((msg) => (
              <div key={msg.id} className="mb-1">
                <span className="text-gray-500">[{msg.timestamp}]</span>{' '}
                <span className={
                  msg.type === 'success' ? 'text-green-400' :
                  msg.type === 'error' ? 'text-red-400' :
                  msg.type === 'warning' ? 'text-yellow-400' :
                  'text-blue-400'
                }>
                  {msg.text}
                </span>
              </div>
            ))}
            {messages.length === 0 && (
              <div className="text-gray-500">Waiting for activity...</div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

export default App;