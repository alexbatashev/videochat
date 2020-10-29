// e2e/runWithSelenium.ts
import webdriver from "selenium-webdriver";
import assert from "assert";
import path from "path";

let driver: webdriver.WebDriver;

beforeAll(async () => {
  let capabilities: webdriver.Capabilities;
  switch (process.env.BROWSER || "chrome") {
    case "chrome": {
      require("chromedriver");
      capabilities = webdriver.Capabilities.chrome();
      capabilities.set("chromeOptions", {
        args: [
          "--headless",
          "--no-sandbox",
          "--disable-gpu",
          "--window-size=1980,1200"
        ]
      });
      break;
    }
  }
  driver = await new webdriver.Builder()
    .withCapabilities(capabilities)
    .build();
});

afterAll(async () => {
  await driver.quit()
});

function delay(ms: number) {
  return new Promise( resolve => setTimeout(resolve, ms) );
}

it("E2E", async () => {
  await driver.get('http://localhost/e2e.html');
  assert.strictEqual(await driver.getTitle(), "E2E");
  await driver.findElement(webdriver.By.id("start_btn")).click()
  await delay(300);
  await driver.findElement(webdriver.By.id("stop_btn")).click()
});
