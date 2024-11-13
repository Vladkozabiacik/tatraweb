function toggleWorksiteField() {
    var position = document.getElementById("position").value;
    var worksiteDiv = document.getElementById("worksiteDiv");

    if (position === "worker") {
        worksiteDiv.classList.remove("hidden");
    } else {
        worksiteDiv.classList.add("hidden");
    }
}
