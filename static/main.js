document.addEventListener('DOMContentLoaded', function () {
    // в каждой таблице кликаем по <th> для сортировки
    document.querySelectorAll('table').forEach(function (table) {
        table.querySelectorAll('th').forEach(function (header, index) {
            header.addEventListener('click', function () {
                sortTableByColumn(table, index);
            });
        });
    });

    function sortTableByColumn(table, column) {
        const rows = Array.from(table.rows).slice(1);  // пропускаем заголовок
        const asc  = table.asc = !table.asc;           // чередуем направление

        rows.sort(function (rowA, rowB) {
            const a = rowA.cells[column].innerText.trim();
            const b = rowB.cells[column].innerText.trim();

            const cmp = (!isNaN(a) && !isNaN(b))
                ? (Number(a) - Number(b))              // числовая колонка
                : a.localeCompare(b);                  // текстовая

            return asc ? cmp : -cmp;
        });

        rows.forEach(row => table.tBodies[0].appendChild(row));
    }
});
