const video_ratio = $input.first().json.video_ratio
let width = 0;
let height = 0;
if (video_ratio === '1:1') {
  width = 2048;
  height = 2048;
} else if (video_ratio === '4:3') {
  width = 2304;
  height = 1728;
} else if (video_ratio === '3:4') {
  width = 1728;
  height = 2304;
} else if (video_ratio === '3:2') {
  width = 2496;
  height = 1664;
} else if (video_ratio === '2:3') {
  width = 1664;
  height = 2496;
} else if (video_ratio === '16:9') {
  width = 2560;
  height = 1440;
} else if (video_ratio === '9:16') {
  width = 1440;
  height = 2560;
} else if (video_ratio === '21:9') {
  width = 3024;
  height = 1296;
} else if (video_ratio === '9:21') {
  width = 1296;
  height = 3024;
}

return {
  "video_width": width,
  "video_height": height,
}