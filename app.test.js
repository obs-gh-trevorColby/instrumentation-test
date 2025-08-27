const request = require('supertest');
const app = require('./dist/app').default;

describe('Express App', () => {
  test('GET / should return welcome message', async () => {
    const response = await request(app).get('/');
    expect(response.status).toBe(200);
    expect(response.body.message).toBe('Welcome to the Instrumentation Test API');
  });

  test('GET /health should return health status', async () => {
    const response = await request(app).get('/health');
    expect(response.status).toBe(200);
    expect(response.body.status).toBe('healthy');
    expect(response.body.timestamp).toBeDefined();
  });

  test('GET /users should return users array', async () => {
    const response = await request(app).get('/users');
    expect(response.status).toBe(200);
    expect(Array.isArray(response.body)).toBe(true);
    expect(response.body.length).toBeGreaterThan(0);
  });

  test('GET /users/:id should return specific user', async () => {
    const response = await request(app).get('/users/1');
    expect(response.status).toBe(200);
    expect(response.body.id).toBe(1);
    expect(response.body.name).toBeDefined();
  });

  test('GET /users/:id should return 404 for non-existent user', async () => {
    const response = await request(app).get('/users/999');
    expect(response.status).toBe(404);
    expect(response.body.error).toBe('User not found');
  });

  test('POST /users should create new user', async () => {
    const newUser = { name: 'Test User', email: 'test@example.com' };
    const response = await request(app)
      .post('/users')
      .send(newUser);
    
    expect(response.status).toBe(201);
    expect(response.body.name).toBe(newUser.name);
    expect(response.body.email).toBe(newUser.email);
    expect(response.body.id).toBeDefined();
  });

  test('POST /users should return 400 for missing data', async () => {
    const response = await request(app)
      .post('/users')
      .send({ name: 'Test User' }); // missing email
    
    expect(response.status).toBe(400);
    expect(response.body.error).toBe('Name and email are required');
  });

  test('GET /error should return 500', async () => {
    const response = await request(app).get('/error');
    expect(response.status).toBe(500);
    expect(response.body.error).toBeDefined();
  });

  test('GET /nonexistent should return 404', async () => {
    const response = await request(app).get('/nonexistent');
    expect(response.status).toBe(404);
    expect(response.body.error).toBe('Route not found');
  });
});
