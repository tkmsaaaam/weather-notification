BASE_URL = 'https://weather.tsukumijima.net/api/forecast/city/'
require 'slack-ruby-client'
require 'faraday'

Slack.configure.token = ARGV[1]

slack = Slack::Web::Client.new

url = BASE_URL + ARGV[0]
res_raw = Faraday.get(url).body
res = JSON.parse(res_raw)

public_time_formatted = res['publicTimeFormatted']
text = res['description']['text']
forecasts = res['forecasts']
forecast = forecasts.find{ |f| f['dateLabel'] == '今日' }
min_temp = forecast['temperature']['min']['celsius']
max_temp = forecast['temperature']['max']['celsius']
chance_of_rains = forecast['chanceOfRain'].map { |k, v| "#{k}: #{v}"  }.join("\n")

message = "日時:#{public_time_formatted}\n概要:#{text}\n最低気温:#{min_temp}\n最高気温:#{max_temp}\n#{chance_of_rains}"

slack.chat_postMessage(channel: ARGV[2], text: message)
