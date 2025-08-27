const express = require('express');
const axios = require('axios');

const app = express();
const PORT = process.env.PORT || 3000;

// Middleware
app.use(express.json());

// In-memory data store
let users = [
  { id: 1, name: 'John Doe', email: 'john@example.com' },
  { id: 2, name: 'Jane Smith', email: 'jane@example.com' }
];

// Routes
app.get('/', (req, res) => {
  res.json({ message: 'Welcome to the Instrumentation Test API' });
});

app.get('/health', (req, res) => {
  res.json({ status: 'healthy', timestamp: new Date().toISOString() });
});

app.get('/users', (req, res) => {
  console.log('Fetching all users');
  res.json(users);
});

app.get('/users/:id', (req, res) => {
  const userId = parseInt(req.params.id);
  console.log(`Fetching user with ID: ${userId}`);
  
  const user = users.find(u => u.id === userId);
  if (!user) {
    return res.status(404).json({ error: 'User not found' });
  }
  
  res.json(user);
});

app.post('/users', (req, res) => {
  const { name, email } = req.body;
  
  if (!name || !email) {
    return res.status(400).json({ error: 'Name and email are required' });
  }
  
  const newUser = {
    id: users.length + 1,
    name,
    email
  };
  
  users.push(newUser);
  console.log(`Created new user: ${JSON.stringify(newUser)}`);
  
  res.status(201).json(newUser);
});

app.get('/external-api', async (req, res) => {
  try {
    console.log('Making external API call');
    const response = await axios.get('https://jsonplaceholder.typicode.com/posts/1');
    res.json(response.data);
  } catch (error) {
    console.error('External API call failed:', error.message);
    res.status(500).json({ error: 'Failed to fetch external data' });
  }
});

app.get('/slow', (req, res) => {
  console.log('Processing slow request');
  setTimeout(() => {
    res.json({ message: 'This was a slow operation', duration: '2 seconds' });
  }, 2000);
});

app.get('/error', (req, res) => {
  console.error('Intentional error triggered');
  res.status(500).json({ error: 'This is an intentional error for testing' });
});

// Error handling middleware
app.use((err, req, res, next) => {
  console.error('Unhandled error:', err);
  res.status(500).json({ error: 'Internal server error' });
});

// 404 handler
app.use((req, res) => {
  res.status(404).json({ error: 'Route not found' });
});

app.listen(PORT, () => {
  console.log(`Server is running on port ${PORT}`);
});

module.exports = app;
