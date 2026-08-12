// Client Application State
let state = {
    rooms: [],            // List of room objects: { id, external_user_id, name }
    activeRoomId: null,   // Active room ID
    isGenerating: false   // Loading state
};

// Selectors
const el = {
    newRoomBtn: document.getElementById('new-room-btn'),
    roomsList: document.getElementById('rooms-list'),
    activeRoomTitle: document.getElementById('active-room-title'),
    activeRoomIdBadge: document.getElementById('active-room-id-badge'),
    clearRoomBtn: document.getElementById('clear-room-btn'),
    messagesContainer: document.getElementById('messages-container'),
    welcomeScreen: document.getElementById('welcome-screen'),
    chatMessages: document.getElementById('chat-messages'),
    typingIndicator: document.getElementById('typing-indicator'),
    chatForm: document.getElementById('chat-form'),
    userInput: document.getElementById('user-input'),
    sendBtn: document.getElementById('send-btn'),
    contextTopic: document.getElementById('context-topic'),
    contextSummary: document.getElementById('context-summary'),
    contextState: document.getElementById('context-state'),
    contextMemories: document.getElementById('context-memories'),
    strategySelect: document.getElementById('strategy-select'),
    maxTokensSlider: document.getElementById('max-tokens-slider'),
    maxTokensVal: document.getElementById('max-tokens-val'),
    policyStrategyBadge: document.getElementById('policy-strategy-badge'),
    tokensEstimatedBadge: document.getElementById('tokens-estimated-badge'),
    contextReferences: document.getElementById('context-references'),
    toast: document.getElementById('toast')
};

// Initial setup on DOM Load
document.addEventListener('DOMContentLoaded', () => {
    init();
});

function init() {
    // 1. Load rooms from Local Storage
    const savedRooms = localStorage.getItem('contextkit_rooms');
    const savedActiveRoom = localStorage.getItem('contextkit_active_room');
    
    if (savedRooms) {
        try {
            state.rooms = JSON.parse(savedRooms);
        } catch (e) {
            state.rooms = [];
        }
    }
    
    // 2. Set event listeners
    el.newRoomBtn.addEventListener('click', createNewRoom);
    el.chatForm.addEventListener('submit', handleFormSubmit);
    el.clearRoomBtn.addEventListener('click', resetCurrentRoom);
    if (el.maxTokensSlider && el.maxTokensVal) {
        el.maxTokensSlider.addEventListener('input', (e) => {
            el.maxTokensVal.textContent = `${e.target.value} tokens`;
        });
    }
    
    // Set up click listeners for demo prompts
    document.querySelectorAll('.demo-prompt-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            if (state.activeRoomId && !state.isGenerating) {
                el.userInput.value = e.target.textContent;
                el.userInput.focus();
            } else if (!state.activeRoomId) {
                showToast("Please create or select a Trip Session first!", "error");
            }
        });
    });
    
    // 3. Render rooms and load active
    renderRooms();
    
    if (savedActiveRoom && state.rooms.some(r => r.id === savedActiveRoom)) {
        switchRoom(savedActiveRoom);
    } else if (state.rooms.length > 0) {
        switchRoom(state.rooms[0].id);
    } else {
        showWelcomeScreen();
    }
}

// Show/Hide Welcome Screen
function showWelcomeScreen() {
    state.activeRoomId = null;
    localStorage.removeItem('contextkit_active_room');
    
    el.welcomeScreen.style.display = 'flex';
    el.chatMessages.style.display = 'none';
    el.chatMessages.innerHTML = '';
    
    el.activeRoomTitle.textContent = 'Select or Create a Trip';
    el.activeRoomIdBadge.textContent = 'No active session';
    el.activeRoomIdBadge.className = 'badge';
    
    el.userInput.disabled = true;
    el.userInput.placeholder = 'Create a Trip Session to start planning...';
    el.sendBtn.disabled = true;
    el.clearRoomBtn.disabled = true;
    
    updateContextPanel(null);
}

// Toast Notifications
function showToast(message, type = "success") {
    el.toast.textContent = message;
    el.toast.className = `toast show ${type}`;
    
    setTimeout(() => {
        el.toast.classList.remove('show');
    }, 3000);
}

// REST: Create a New Room
async function createNewRoom() {
    try {
        el.newRoomBtn.disabled = true;
        const timestamp = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
        const roomName = `Trip Session (${timestamp})`;
        
        const response = await fetch('/api/rooms/new', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ external_user_id: `web_user_${Date.now()}` })
        });
        
        if (!response.ok) throw new Error("Failed to create room");
        
        const room = await response.json();
        
        const newRoomObj = {
            id: room.id,
            external_user_id: room.external_user_id,
            name: roomName
        };
        
        state.rooms.unshift(newRoomObj);
        localStorage.setItem('contextkit_rooms', JSON.stringify(state.rooms));
        
        renderRooms();
        switchRoom(room.id);
        showToast("New trip session created!");
    } catch (err) {
        showToast("Error creating trip session: " + err.message, "error");
    } finally {
        el.newRoomBtn.disabled = false;
    }
}

// Switch Active Room
async function switchRoom(roomId) {
    state.activeRoomId = roomId;
    localStorage.setItem('contextkit_active_room', roomId);
    
    // Update active UI elements in sidebar list
    document.querySelectorAll('.room-item').forEach(item => {
        if (item.dataset.id === roomId) {
            item.classList.add('active');
        } else {
            item.classList.remove('active');
        }
    });
    
    const roomObj = state.rooms.find(r => r.id === roomId);
    el.activeRoomTitle.textContent = roomObj ? roomObj.name : 'Active Trip';
    el.activeRoomIdBadge.textContent = `Room ID: ${roomId}`;
    el.activeRoomIdBadge.className = 'badge badge-room-id';
    
    el.userInput.disabled = false;
    el.userInput.placeholder = 'Ask the agent to look up hotels, flights, or activities...';
    el.sendBtn.disabled = false;
    el.clearRoomBtn.disabled = false;
    
    el.welcomeScreen.style.display = 'none';
    el.chatMessages.style.display = 'flex';
    el.chatMessages.innerHTML = '';
    
    // Load state and messages from server
    await loadRoomState(roomId);
}

// Fetch and load room details
async function loadRoomState(roomId) {
    try {
        setGenerating(true);
        const response = await fetch(`/api/rooms/${roomId}/state`);
        if (!response.ok) throw new Error("Failed to retrieve room state");
        
        const data = await response.json();
        
        // 1. Render message history
        const messages = data.messages || [];
        if (messages.length === 0) {
            el.chatMessages.innerHTML = `<div class="empty-list-state" style="border:none; margin: 40px auto; max-width: 320px;">No messages yet. Start chatting below!</div>`;
        } else {
            el.chatMessages.innerHTML = '';
            messages.forEach(msg => {
                appendMessage(msg.role, msg.content);
            });
        }
        
        // 2. Update context panel
        updateContextPanel(data.context);
        
        // 3. Rename room locally if there is an active topic/destination
        if (data.context && data.context.active_topic) {
            updateRoomNameLocally(roomId, data.context.active_topic, data.context.summary);
        }
        
    } catch (err) {
        showToast("Error loading conversation: " + err.message, "error");
    } finally {
        setGenerating(false);
        scrollToBottom();
    }
}

// Reset room state / messages
async function resetCurrentRoom() {
    if (!state.activeRoomId) return;
    if (!confirm("Are you sure you want to reset this trip session? This deletes all conversation history and state.")) return;
    
    try {
        setGenerating(true);
        const response = await fetch(`/api/rooms/${state.activeRoomId}`, {
            method: 'DELETE'
        });
        
        if (!response.ok) throw new Error("Delete failed");
        
        // Remove room from current local list
        state.rooms = state.rooms.filter(r => r.id !== state.activeRoomId);
        localStorage.setItem('contextkit_rooms', JSON.stringify(state.rooms));
        
        renderRooms();
        showWelcomeScreen();
        showToast("Session reset and deleted successfully.");
    } catch (err) {
        showToast("Reset session failed: " + err.message, "error");
    } finally {
        setGenerating(false);
    }
}

// Update Room Name based on Active Topic/Context
function updateRoomNameLocally(roomId, activeTopic, summary) {
    const roomIdx = state.rooms.findIndex(r => r.id === roomId);
    if (roomIdx === -1) return;
    
    let generatedName = "";
    if (activeTopic && activeTopic.trim() !== "") {
        // Clean topic string
        generatedName = activeTopic.charAt(0).toUpperCase() + activeTopic.slice(1);
    } else if (summary && summary.trim() !== "") {
        // Truncate summary into a short title
        const summaryWords = summary.split(" ");
        generatedName = summaryWords.slice(0, 3).join(" ") + "...";
    }
    
    if (generatedName && state.rooms[roomIdx].name !== generatedName) {
        state.rooms[roomIdx].name = generatedName;
        localStorage.setItem('contextkit_rooms', JSON.stringify(state.rooms));
        
        // Refresh UI
        renderRooms();
        if (state.activeRoomId === roomId) {
            el.activeRoomTitle.textContent = generatedName;
        }
    }
}

// Render Sidebar Room Items
function renderRooms() {
    if (state.rooms.length === 0) {
        el.roomsList.innerHTML = `<div class="empty-list-state">No active sessions</div>`;
        return;
    }
    
    el.roomsList.innerHTML = '';
    state.rooms.forEach(room => {
        const li = document.createElement('li');
        li.className = `room-item ${room.id === state.activeRoomId ? 'active' : ''}`;
        li.dataset.id = room.id;
        
        const itemContent = document.createElement('div');
        itemContent.className = 'room-item-content';
        itemContent.innerHTML = `
            <i class="fa-solid fa-map-location room-icon"></i>
            <span class="room-name">${escapeHTML(room.name)}</span>
        `;
        
        const deleteBtn = document.createElement('button');
        deleteBtn.className = 'room-delete-btn';
        deleteBtn.innerHTML = `<i class="fa-solid fa-xmark"></i>`;
        deleteBtn.title = "Delete session";
        deleteBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            if (confirm(`Delete session "${room.name}"?`)) {
                deleteRoom(room.id);
            }
        });
        
        li.appendChild(itemContent);
        li.appendChild(deleteBtn);
        
        li.addEventListener('click', () => {
            switchRoom(room.id);
        });
        
        el.roomsList.appendChild(li);
    });
}

// Delete room logic
async function deleteRoom(roomId) {
    try {
        // Send request to delete room in backend
        await fetch(`/api/rooms/${roomId}`, { method: 'DELETE' });
    } catch (e) {
        console.error("Backend delete room error: ", e);
    }
    
    state.rooms = state.rooms.filter(r => r.id !== roomId);
    localStorage.setItem('contextkit_rooms', JSON.stringify(state.rooms));
    renderRooms();
    
    if (state.activeRoomId === roomId) {
        if (state.rooms.length > 0) {
            switchRoom(state.rooms[0].id);
        } else {
            showWelcomeScreen();
        }
    }
    showToast("Session deleted.");
}

// Submit Message Event
async function handleFormSubmit(e) {
    e.preventDefault();
    if (!state.activeRoomId || state.isGenerating) return;
    
    const messageText = el.userInput.value.trim();
    if (!messageText) return;
    
    // Clear input
    el.userInput.value = '';
    
    // Remove empty state message if present
    const emptyState = el.chatMessages.querySelector('.empty-list-state');
    if (emptyState) emptyState.remove();
    
    // 1. Append User message to Feed
    appendMessage('user', messageText);
    scrollToBottom();
    
    try {
        setGenerating(true);
        
        // 2. POST Chat turn to server
        const strategy = el.strategySelect ? el.strategySelect.value : 'auto';
        const maxTokens = el.maxTokensSlider ? parseInt(el.maxTokensSlider.value) : 4000;

        const response = await fetch('/api/chat', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                room_id: state.activeRoomId,
                message: messageText,
                strategy: strategy,
                max_tokens: maxTokens
            })
        });
        
        if (!response.ok) throw new Error("Failed to get agent response");
        
        const data = await response.json();
        
        // 3. Render tool-calls (if any)
        if (data.tool_calls && data.tool_calls.length > 0) {
            data.tool_calls.forEach(toolCall => {
                appendToolCard(toolCall.tool, toolCall.input, toolCall.output);
            });
        }
        
        // 4. Append Assistant Response to Feed
        appendMessage('assistant', data.response);
        
        // 5. Update context panel
        updateContextPanel(data.context);
        
        // 6. Check room name update
        if (data.context && data.context.active_topic) {
            updateRoomNameLocally(state.activeRoomId, data.context.active_topic, data.context.summary);
        }
        
    } catch (err) {
        appendMessage('assistant', `⚠️ Error: Could not process request. (${err.message})`);
        showToast("Error processing request: " + err.message, "error");
    } finally {
        setGenerating(false);
        scrollToBottom();
    }
}

// Append Message to Feed
function appendMessage(role, content) {
    const isUser = role === 'user';
    const messageRow = document.createElement('div');
    messageRow.className = `message-row ${isUser ? 'user' : 'assistant'}`;
    
    const avatar = document.createElement('div');
    avatar.className = 'message-avatar';
    avatar.innerHTML = isUser ? '<i class="fa-solid fa-user"></i>' : '<i class="fa-solid fa-robot"></i>';
    
    const msgContent = document.createElement('div');
    msgContent.className = 'message-content';
    
    const bubble = document.createElement('div');
    bubble.className = 'message-bubble';
    bubble.innerHTML = formatMarkdown(content);
    
    const timeSpan = document.createElement('span');
    timeSpan.className = 'message-time';
    timeSpan.textContent = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    
    msgContent.appendChild(bubble);
    msgContent.appendChild(timeSpan);
    
    messageRow.appendChild(avatar);
    messageRow.appendChild(msgContent);
    
    el.chatMessages.appendChild(messageRow);
}

// Append Collapsible Tool Execution Log Card
function appendToolCard(toolName, input, output) {
    const cardContainer = document.createElement('div');
    cardContainer.className = 'tool-card-container';
    
    // Format input and output prettily
    let inputStr = typeof input === 'object' ? JSON.stringify(input, null, 2) : String(input);
    let outputStr = "";
    try {
        // Try parsing output if it is a JSON string to prettify it
        let parsed = JSON.parse(output);
        outputStr = JSON.stringify(parsed, null, 2);
    } catch (e) {
        outputStr = String(output);
    }
    
    const cardId = `tool-card-${Date.now()}-${Math.floor(Math.random() * 1000)}`;
    
    cardContainer.innerHTML = `
        <div class="tool-card" id="${cardId}">
            <div class="tool-card-header">
                <div class="tool-card-title">
                    <i class="fa-solid fa-gears"></i>
                    <span>Tool Log: ${toolName}</span>
                    <span class="tool-badge">Executed</span>
                </div>
                <div class="tool-card-toggle"><i class="fa-solid fa-chevron-down"></i></div>
            </div>
            <div class="tool-card-body">
                <div class="tool-log-item">
                    <div class="tool-log-label">Arguments Passed</div>
                    <pre class="tool-log-value">${escapeHTML(inputStr)}</pre>
                </div>
                <div class="tool-log-item">
                    <div class="tool-log-label">Execution Result</div>
                    <pre class="tool-log-value">${escapeHTML(outputStr)}</pre>
                </div>
            </div>
        </div>
    `;
    
    el.chatMessages.appendChild(cardContainer);
    
    // Add toggle event listener
    const cardEl = cardContainer.querySelector('.tool-card');
    const headerEl = cardContainer.querySelector('.tool-card-header');
    headerEl.addEventListener('click', () => {
        cardEl.classList.toggle('open');
    });
}

// Update Konteks Context Panel UI
function updateContextPanel(context) {
    if (!context) {
        el.contextTopic.textContent = 'No active topic';
        el.contextTopic.className = 'context-value-box empty';
        el.contextSummary.textContent = 'No summary generated yet.';
        el.contextSummary.className = 'context-value-box empty';
        el.contextState.textContent = '{}';
        el.contextState.className = 'context-code empty';
        el.contextMemories.innerHTML = `<div class="empty-memories-state">No long-term memories extracted yet.</div>`;
        if (el.contextReferences) el.contextReferences.innerHTML = `No references resolved yet`;
        if (el.tokensEstimatedBadge) el.tokensEstimatedBadge.textContent = `0 tokens`;
        return;
    }
    
    // 0. Phase 2 Policy & Strategy Metadata
    if (el.policyStrategyBadge && context.strategy_used) {
        el.policyStrategyBadge.textContent = context.strategy_used;
    }
    if (el.tokensEstimatedBadge && context.tokens_estimated !== undefined) {
        el.tokensEstimatedBadge.textContent = `${context.tokens_estimated} tokens`;
    }

    // 0b. Resolved References
    if (el.contextReferences) {
        const refs = context.references || {};
        const keys = Object.keys(refs);
        if (keys.length > 0) {
            el.contextReferences.innerHTML = '';
            el.contextReferences.className = 'references-list';
            keys.forEach(k => {
                const pill = document.createElement('div');
                pill.className = 'reference-pill';
                pill.innerHTML = `<span class="ref-source">"${escapeHTML(k)}"</span> <i class="fa-solid fa-arrow-right-long"></i> <span class="ref-target">${escapeHTML(refs[k])}</span>`;
                el.contextReferences.appendChild(pill);
            });
        } else {
            el.contextReferences.innerHTML = `No references resolved yet`;
            el.contextReferences.className = 'references-list empty';
        }
    }

    // 1. Active Topic
    const topic = context.active_topic;
    if (topic && topic.trim() !== "") {
        el.contextTopic.textContent = topic;
        el.contextTopic.className = 'context-value-box';
    } else {
        el.contextTopic.textContent = 'No active topic';
        el.contextTopic.className = 'context-value-box empty';
    }
    
    // 2. Summary
    const summary = context.summary;
    if (summary && summary.trim() !== "") {
        el.contextSummary.textContent = summary;
        el.contextSummary.className = 'context-value-box';
    } else {
        el.contextSummary.textContent = 'No summary generated yet.';
        el.contextSummary.className = 'context-value-box empty';
    }
    
    // 3. Structured State
    const stateVal = context.state;
    if (stateVal && Object.keys(stateVal).length > 0) {
        el.contextState.textContent = JSON.stringify(stateVal, null, 2);
        el.contextState.className = 'context-code';
    } else {
        el.contextState.textContent = '{}';
        el.contextState.className = 'context-code empty';
    }
    
    // 4. Long-term Memories
    const memories = context.memories || [];
    if (memories.length > 0) {
        el.contextMemories.innerHTML = '';
        memories.forEach(mem => {
            const li = document.createElement('li');
            li.className = 'memory-item';
            li.textContent = mem.content;
            el.contextMemories.appendChild(li);
        });
    } else {
        el.contextMemories.innerHTML = `<div class="empty-memories-state">No long-term memories extracted yet.</div>`;
    }
}

// UI State Helper
function setGenerating(isGen) {
    state.isGenerating = isGen;
    
    if (isGen) {
        el.typingIndicator.style.display = 'flex';
        el.userInput.disabled = true;
        el.sendBtn.disabled = true;
    } else {
        el.typingIndicator.style.display = 'none';
        el.userInput.disabled = false;
        el.sendBtn.disabled = false;
        el.userInput.focus();
    }
}

// Helpers
function scrollToBottom() {
    if (el.messagesContainer) {
        setTimeout(() => {
            el.messagesContainer.scrollTo({
                top: el.messagesContainer.scrollHeight,
                behavior: 'smooth'
            });
            window.scrollTo(0, 0);
        }, 50);
    }
}

function escapeHTML(str) {
    return str
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

function formatMarkdown(text) {
    if (!text) return "";
    let formatted = escapeHTML(text);
    
    // Bold: **text** -> <strong>text</strong>
    formatted = formatted.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
    
    // Inline code: `code` -> <code>code</code>
    formatted = formatted.replace(/`(.*?)`/g, '<code>$1</code>');
    
    // Bullet points: \n- item -> <li>item</li>
    if (formatted.includes('\n- ') || formatted.includes('\n* ')) {
        // Simple list splitting
        const lines = formatted.split('\n');
        let inList = false;
        const processedLines = lines.map(line => {
            const match = line.match(/^(\s*)[-*]\s+(.*)/);
            if (match) {
                let out = "";
                if (!inList) {
                    out += '<ul style="margin: 8px 0 8px 20px; list-style-type: disc;">';
                    inList = true;
                }
                out += `<li>${match[2]}</li>`;
                return out;
            } else {
                let out = "";
                if (inList) {
                    out += '</ul>';
                    inList = false;
                }
                out += line;
                return out;
            }
        });
        if (inList) {
            processedLines.push('</ul>');
        }
        formatted = processedLines.join('<br>');
    } else {
        // Linebreaks
        formatted = formatted.replace(/\n/g, '<br>');
    }
    
    return formatted;
}
