function toTables(keyPrefix, htmlElement, data, excludedKeys) {
  // Filter keys starting with "system."
  const systemKeys = Object.keys(data)
    .filter((key) => key.startsWith(keyPrefix))
    .filter((k) => !excludedKeys.includes(k));

  htmlElement.innerHTML = ""; // Clear any previous rows.

  // Add table headers.
  const headerRow = htmlElement.insertRow();
  headerRow.innerHTML = "<th>Key</th><th></th>";

  // Add rows for each "system." key.
  systemKeys.forEach((key) => {
    const row = htmlElement.insertRow();
    row.innerHTML = `<td>${key.replace(keyPrefix, "")}</td><td>${
      typeof data[key] === "object" ? JSON.stringify(data[key]) : data[key]
    }</td>`;
  });
}

// Populate the filterable table
function populateFilterableTable(table, paths, bucketTimes) {
  table.innerHTML = [
    "<thead><tr>",
    "<th>Path</th>",
    "<th>Active in //</th>",
    "<th>Max Active in //</th>",
    "<th>Total Count</th>",
    "<th>Total Time</th>",
    "<th class='nowrap'>P 50</th>",
    "<th class='nowrap'>P 95</th>",
    "<th class='nowrap'>P 98</th>",
    "<th class='nowrap'>P 99</th>",
  ]
    .concat(bucketTimes.map((t) => "<th> R " + t + "</th>"))
    .concat([
      "<th>Statuses</th>",
      "<th>First seen</th>",
      "<th>Last seen</th>",
      "</tr></thead>",
    ])
    .join(""); // Reset table header
  const tbody = document.createElement("tbody");
  const tbodyElement = table.appendChild(tbody);
  for (let path of Object.keys(paths)) {
    const stats = paths[path];
    const row = tbodyElement.insertRow();
    row.addEventListener("click", () => {
      drawHistogram(bucketTimes, stats["counts"], path);
    });
    row.innerHTML = [
      `<td>${path}</td>`,
      `<td>${stats["active"]}</td>`,
      `<td>${stats["maxActive"]}</td>`,
      `<td>${stats["totalCount"]}</td>`,
      `<td>${stats["totalTime"]}</td>`,
      `<td>${stats["50"]}</td>`,
      `<td>${stats["95"]}</td>`,
      `<td>${stats["98"]}</td>`,
      `<td>${stats["99"]}</td>`,
    ]
      .concat(
        stats["counts"]
          .slice(0, stats["counts"].length - 1)
          .map((c) => "<td>" + (c == 0 ? "" : c) + "</td>")
      )
      .concat([
        `<td>${JSON.stringify(stats["statusCount"])}</td>`,
        `<td>${stats["firstSeen"]}</td>`,
        `<td>${stats["lastSeen"]}</td>`,
      ])
      .join("");
  }
}

// Filter the table based on search input
function filterTable() {
  const searchValue = document
    .getElementById("search-path")
    .value.toLowerCase();
  const table = document.getElementById("info-percentiles-by-path");
  const rows = Array.from(table.rows).slice(1); // Skip the header row

  rows.forEach((row) => {
    const path = row.cells[0].textContent.toLowerCase();
    row.style.display = path.includes(searchValue) ? "" : "none";
  });
}

function drawHistogram(bucketTimes, counts, title) {
  // Maximum bar length
  const maxBarLength = 50; // Adjust this for wider bars
  const maxCount = Math.max(...counts);
  // Pad seconds to the same width
  const paddedTimes = bucketTimes.map((time) =>
    time.toFixed(2).padStart(6, " ")
  );

  // Generate the ASCII bar chart
  let chart = paddedTimes
    .map((time, index) => {
      const barLength = Math.round((counts[index] / maxCount) * maxBarLength);
      const bar = "#".repeat(barLength);
      return `${time} s | ${bar} ${counts[index]}`;
    })
    .join("\n");

  // Render the chart in the pre tag
  document.getElementById("ascii-chart").textContent = title + "\n" + chart;
}

async function fetchAndDisplayInfo() {
  try {
    const response = await fetch("/__banme/api/info");
    if (!response.ok) {
      throw new Error(`HTTP error! Status: ${response.status}`);
    }
    const data = await response.json();

    const infoBannedElement = document.getElementById("info-banned");
    infoBannedElement.textContent = JSON.stringify(data.banned, null, 2);

    toTables("system.", document.getElementById("info-system"), data, []);
    toTables(
      "percentiles.",
      document.getElementById("info-percentiles"),
      data,
      ["percentiles.byPath"]
    );
    const bucketTimes = data["percentiles.buckets"];

    drawHistogram(bucketTimes, data["percentiles.bucketCounts"], "general");

    // Attach event listener for the search field
    document
      .getElementById("search-path")
      .addEventListener("input", filterTable);

    populateFilterableTable(
      document.getElementById("info-percentiles-by-path"),
      data["percentiles.byPath"],
      bucketTimes
    );
    filterTable();

    const ipsElement = document.getElementById("info-status-by-ip");

    const allStatuses = new Set();
    for (let ip of Object.keys(data.statusCountPerIp)) {
      const stats = data.statusCountPerIp[ip];
      for (let status of Object.keys(stats)) {
        allStatuses.add(status);
      }
    }

    
    const statuses = Array.from(allStatuses);
    statuses.sort();
    const columns = ["ip"].concat(statuses).concat(["Last seen", "Links"]);

    ipsElement.innerHTML =
      "<thead><tr>" +
      columns.map((s) => "<th>" + s + "</th>").join(" ") +
      "</tr></thead>";

    const listOfIps = Object.keys(data.statusCountPerIp).sort((a, b) => {
      const num1 = Number(
        a
          .split(".")
          .map((num) => `000${num}`.slice(-3))
          .join("")
      );
      const num2 = Number(
        b
          .split(".")
          .map((num) => `000${num}`.slice(-3))
          .join("")
      );
      return num1 - num2;
    });
    const tbody = document.createElement("tbody");
    const ipsElementTbody = ipsElement.appendChild(tbody);
    for (let ip of listOfIps) {
      const stats = data.statusCountPerIp[ip];

      const row = ipsElementTbody.insertRow();
      const others = statuses
        .map((s) => stats[s])
        .map((r) => `<td>${r == undefined ? "" : r}</td>`);
      row.innerHTML = `<td>${ip}</td>${others.join("")}<td>${data["lastSeen"][ip]}</td>
      <td><a href='https://ipinfo.io/${ip}'>ipinfo</a> <a href='https://www.abuseipdb.com/check/${ip}'>abuseip</a></td>`;
    }

    // Display the data in an element with ID 'info'.
    const infoElement = document.getElementById("banme-info");
    infoElement.textContent = JSON.stringify(data, null, 2);

    makeTablesSortable();
  } catch (error) {
    console.error("Error fetching info:", error);
  }
}

// Generic table sorting function
function makeTablesSortable() {
  const tables = document.querySelectorAll(".sortable");

  tables.forEach((table) => {
    const headers = table.querySelectorAll("th");
    headers.forEach((header, columnIndex) => {
      header.addEventListener("click", () => {
        sortTable(table, columnIndex);
        updateSortIndicators(header, headers);
      });
    });
  });
}

function sortTable(table, columnIndex) {
  const tbody = table.tBodies[0];
  const rows = Array.from(tbody.rows);

  // Determine the current sorting order
  const currentOrder = table.dataset.sortOrder || "asc";
  const isAscending = currentOrder === "desc";
  table.dataset.sortOrder = isAscending ? "asc" : "desc";

  // Sort rows
  rows.sort((rowA, rowB) => {
    const cellA = rowA.cells[columnIndex].textContent.trim();
    const cellB = rowB.cells[columnIndex].textContent.trim();

    // Numeric sorting if both cells are numbers
    if (!isNaN(cellA) && !isNaN(cellB)) {
      return isAscending ? cellA - cellB : cellB - cellA;
    }

    // Text sorting
    return isAscending
      ? cellA.localeCompare(cellB)
      : cellB.localeCompare(cellA);
  });

  // Append sorted rows back to the tbody
  rows.forEach((row) => tbody.appendChild(row));
}

function updateSortIndicators(clickedHeader, allHeaders) {
  allHeaders.forEach((header) =>
    header.classList.remove("sort-asc", "sort-desc")
  );
  clickedHeader.classList.add(
    clickedHeader.parentElement.parentElement.parentElement.dataset
      .sortOrder === "asc"
      ? "sort-asc"
      : "sort-desc"
  );
}

// Refresh the info every 5 seconds.
setInterval(() => {
  const checkbox = document.getElementById("autorefresh");
  if (!checkbox.checked) {
    return;
  }
  fetchAndDisplayInfo();
}, 5000);

// Initial load.
fetchAndDisplayInfo();

const currentDomain = window.location.hostname; // For domain name only
const currentURL = window.location.href; // For full URL

// Set the document title
document.title = "Ban Me - " + currentDomain; // Use `currentURL` if you want the full URL
