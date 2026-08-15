// PM2 config untuk KasQ frontend (SvelteKit adapter-node).
// __KASQ_ORIGIN__ diganti otomatis saat make deploy (dari DEPLOY_PUBLIC_URL).

module.exports = {
	apps: [
		{
			name: 'kasq-fe',
			cwd: __dirname,
			script: './build/index.js',
			instances: 1,
			exec_mode: 'fork',
			autorestart: true,
			max_restarts: 10,
			env: {
				NODE_ENV: 'production',
				PORT: 3008,
				HOST: '127.0.0.1',
				ORIGIN: '__KASQ_ORIGIN__',
				PROTOCOL_HEADER: 'x-forwarded-proto',
				HOST_HEADER: 'x-forwarded-host',
				BODY_SIZE_LIMIT: '10485760'
			}
		}
	]
};
