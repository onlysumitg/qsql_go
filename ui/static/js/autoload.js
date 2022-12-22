 // https://jsfiddle.net/bc_rikko/gbpw2q9x/3/

 function isElementInViewport(el) {
     var rect = el.getBoundingClientRect();

     return rect.bottom > 0 &&
         rect.right > 0 &&
         rect.left < (window.innerWidth || document.documentElement.clientWidth) /* or $(window).width() */ &&
         rect.top < (window.innerHeight || document.documentElement.clientHeight) /* or $(window).height() */ ;
 }


 function handler() {
     const el2 = document.getElementById('loadmore')
     console.log("calliong hnadler 2")


     if (typeof (el2) != 'undefined' && el2 != null) {

         console.log("calliong hnadler " + el2 + " " + isElementInViewport(el2))

         if (isElementInViewport(el2)) {

             el2.click();

         }
     }
 }

 




 


 function isElementVisible(el) {
     var rect = el.getBoundingClientRect(),
         vWidth = window.innerWidth || document.documentElement.clientWidth,
         vHeight = window.innerHeight || document.documentElement.clientHeight,
         efp = function (x, y) {
             return document.elementFromPoint(x, y)
         };

     // Return false if it's not in the viewport
     if (rect.right < 0 || rect.bottom < 0 ||
         rect.left > vWidth || rect.top > vHeight)
         return false;

     // Return true if any of its four corners are visible
     return (
         el.contains(efp(rect.left, rect.top)) ||
         el.contains(efp(rect.right, rect.top)) ||
         el.contains(efp(rect.right, rect.bottom)) ||
         el.contains(efp(rect.left, rect.bottom))
     );
 }







 $(document).ready(function () {
     console.log("ready!");
     document.getElementById("resulttable2").addEventListener("scroll", handler);



 });