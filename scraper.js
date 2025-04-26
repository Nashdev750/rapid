const puppeteer = require('puppeteer');
const Redis = require('ioredis');

// Connect to Redis
const redis = new Redis({
  host: 'localhost', // or your VPS IP
  port: 6379,
});

(async () => {
  const url = 'https://footballpredictions.com/footballpredictions/'; // Replace with real page URL

  const browser = await puppeteer.launch(
    {
        headless: false, 
        defaultViewport: {
          width: 1920,
          height: 1080
        },
        args: [
          '--start-maximized'
        ]
      }
  );
  const page = await browser.newPage();
  await page.goto(url, { waitUntil: 'domcontentloaded' });
  console.log('load')
  // ✅ Wait for at least one prediction div to appear
  await page.waitForSelector('.prediction', { timeout: 10000 });
  console.log('load2')
  const predictions = await page.$$eval('.prediction', (nodes) => {
    return nodes.map(node => {
      const homeTeam = node.querySelector('.predictionteams div:nth-child(1) p a')?.textContent.trim();
      const awayTeam = node.querySelector('.predictionteams div:nth-child(2) p a')?.textContent.trim();
      const predictionScore = node.querySelector('.predictionbox strong')?.textContent.trim();
      const dateTimeRaw = node.querySelector('.bp-details p:nth-of-type(2)')?.textContent.trim();

      // Parse match datetime
      const dateMatch = dateTimeRaw?.match(/(\d{2})\/(\d{2})\/(\d{4})/);
      const timeMatch = dateTimeRaw?.match(/at\s(\d{2}:\d{2})/);
      let timestamp = null;
      if (dateMatch && timeMatch) {
        const [_, day, month, year] = dateMatch;
        timestamp = new Date(`${year}-${month}-${day}T${timeMatch[1]}:00Z`).toISOString();
      }

      // Parse predicted scores
      const [homeGoalsStr, awayGoalsStr] = predictionScore ? predictionScore.split('-') : ['0', '0'];
      const homeGoals = parseInt(homeGoalsStr, 10) || 0;
      const awayGoals = parseInt(awayGoalsStr, 10) || 0;
      const totalGoals = homeGoals + awayGoals;

      // Compute derived predictions
      const prediction = {
        match_id: `${homeTeam?.replace(/\s+/g, '')}_vs_${awayTeam?.replace(/\s+/g, '')}`,
        home_team: homeTeam,
        away_team: awayTeam,
        '1x2': homeGoals > awayGoals ? '1' : homeGoals < awayGoals ? '2' : 'X',
        over_under_3_5g: totalGoals > 3 ? 'Over' : 'Under',
        over_under_2_5g: totalGoals > 2 ? 'Over' : 'Under',
        btts: homeGoals > 0 && awayGoals > 0 ? 'Yes' : 'No',
        away_over_under_1_5: awayGoals > 1 ? 'Over' : 'Under',
        away_to_score: awayGoals > 0 ? 'Yes' : 'No',
        home_over_under_1_5: homeGoals > 1 ? 'Over' : 'Under',
        home_to_score: homeGoals > 0 ? 'Yes' : 'No',
        timestamp: timestamp,
      };

      return prediction;
    });
  });
console.log(predictions)
  // Save to Redis
  await redis.set('predictions:today', JSON.stringify(predictions));

  console.log(`✅ Scraped and saved ${predictions.length} matches to Redis!`);

  await browser.close();
  redis.disconnect();
})();
