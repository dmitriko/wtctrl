#!/bin/sh
 curl \
    --header "Ocp-Apim-Subscription-Key: ${SPEECH2TEXT_KEY}" \
    --header "Content-type: audio/wav; codecs=audio/pcm; samplerate=48000" \
    --data-binary @${1} \
   "https://eastus.stt.speech.microsoft.com/speech/recognition/conversation/cognitiveservices/v1?language=ru-RU" 


#    -X POST \
