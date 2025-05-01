const puppeteer = require('puppeteer');
const Redis = require('ioredis');
const { v4: uuidv4 } = require('uuid');

const redis = new Redis({
  host: 'localhost',
  port: 6379,
  password: '43RQ4R45TTTTTT52msh35d1945895fb417p12&565$IYU*776$'
});

(async () => {
  const url = 'https://footballpredictions.com/footballpredictions/';

  const browser = await puppeteer.launch({
    headless: true,
    defaultViewport: { width: 1920, height: 1080 },
    args: ['--start-maximized','--no-sandbox', '--disable-setuid-sandbox']
  });

  const page = await browser.newPage();
  await page.setUserAgent('Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36');
  await page.goto(url, { waitUntil: 'domcontentloaded' });
  await page.waitForSelector('.acc-content.active', { timeout: 10000 });

  // Extract matches
  const matches = await page.$$eval('.acc-content.active script[type="application/ld+json"]', (scripts) => {
    return scripts.map((script) => JSON.parse(script.innerText));
  });

  // Extract predictions
  const predictions = await page.$$eval('.divTableCell2.pred-box-info-result strong', (elements) => {
    return elements.map((el) => el.innerText.trim());
  });

  // Extract dates and times
  const dates = await page.$$eval('.pred-box-info-time .datum', els => els.map(el => el.textContent.trim()));
  const times = await page.$$eval('.pred-box-info-time .tijd', els => els.map(el => el.textContent.trim()));

  const todayPredictions = [];

  for (let i = 0; i < matches.length; i++) {
    const match = matches[i];
    const predictionScore = predictions[i] || "0-0";
    const [homeGoals, awayGoals] = predictionScore.split('-').map(Number);

    // Parse date and time correctly
    const [day, month, year] = dates[i].split('/').map(Number); // e.g., "26/04/2025"
    const [hour, minute] = times[i].split(':').map(Number);     // e.g., "13:30"

    const matchDate = new Date(Date.UTC(year, month - 1, day, hour, minute));

    const prediction = {
      match_id: uuidv4(),
      home_team: match.competitor[0].name.trim(),
      away_team: match.competitor[1].name.trim(),
      '1x2': homeGoals > awayGoals ? "1" : homeGoals < awayGoals ? "2" : "X",
      over_under_3_5g: (homeGoals + awayGoals) > 3.5 ? "Over" : "Under",
      over_under_2_5g: (homeGoals + awayGoals) > 2.5 ? "Over" : "Under",
      btts: homeGoals > 0 && awayGoals > 0 ? "Yes" : "No",
      away_over_under_1_5: awayGoals > 1.5 ? "Over" : "Under",
      away_to_score: awayGoals > 0 ? "Yes" : "No",
      home_over_under_1_5: homeGoals > 1.5 ? "Over" : "Under",
      home_to_score: homeGoals > 0 ? "Yes" : "No",
      timestamp: matchDate.toISOString()
    };

    todayPredictions.push(prediction);

    console.log(`✅ Prediction for ${prediction.home_team} vs ${prediction.away_team} prepared.`);
  }

  // Save to Redis
  await redis.set('predictions:today', JSON.stringify(todayPredictions));

  console.log('🎯 All predictions saved under predictions:today');

  await browser.close();
  await redis.quit();
})();
