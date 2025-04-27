const http = require('https');

const options = {
	method: 'GET',
	hostname: 'football-prediction-api.p.rapidapi.com',
	port: null,
	path: '/predictions',
	headers: {
		'x-rapidapi-key': '94385189ffmshd8e487ba84ce2f1p1b6dd3jsn2739a910c13b',
		'x-rapidapi-host': 'football-prediction-api.p.rapidapi.com'
	}
};

const req = http.request(options, function (res) {
	const chunks = [];

	res.on('data', function (chunk) {
		chunks.push(chunk);
	});

	res.on('end', function () {
		const body = Buffer.concat(chunks);
		console.log(body.toString());
	});
});

req.end();